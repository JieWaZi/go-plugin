package middleware

import (
	"context"
	"fmt"
	"github.com/JieWazi/goplugin/http_trigger/entity"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	"github.com/SkyAPM/go2sky/reporter"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type SkyTracer struct {
	Endpoint     string `yaml:"endpoint"`
	ServiceName  string `yaml:"service_name"`
	functionName string
	tracer       *go2sky.Tracer
	reporter     go2sky.Reporter

	response http.ResponseWriter
	request  *http.Request
	httpFunc HttpFunc

	context    *Context
	tool       *FunctionTool
	spiderFunc SpiderFunc
}

func (s *SkyTracer) Init() error {
	var (
		err error
	)
	if s.Endpoint == "" {
		s.reporter, err = entity.NewLogReporter()
	} else {
		s.reporter, err = reporter.NewGRPCReporter(s.Endpoint)
	}
	if err != nil {
		log.Printf("new reporter error:%v \n", err.Error())
		return err
	}
	s.tracer, err = go2sky.NewTracer(s.ServiceName, go2sky.WithReporter(s.reporter))
	if err != nil {
		log.Printf("NewTracer error:%v \n", err.Error())
		return err
	}
	return nil
}

func (s *SkyTracer) WithSpiderFunc(spiderFunc SpiderFunc, jctx *JsonContext, tool *FunctionTool) {
	s.spiderFunc = spiderFunc
	s.tool = tool
	ctx := jctx.Context
	ctxMap := jctx.ContextMap
	if ctx == nil {
		ctx = context.Background()
	}
	if ctxMap == nil {
		ctxMap = make(map[string]interface{})
	}
	s.context = NewContext(ctx, ctxMap)
}

func (s *SkyTracer) WithHttpFunc(httpFunc HttpFunc, resp http.ResponseWriter, req *http.Request) {
	s.httpFunc = httpFunc
	s.request = req
	s.response = resp
}

func (s *SkyTracer) invokeFunction() error {
	if s.httpFunc != nil {
		s.httpFunc(s.response, s.request)
	} else if s.spiderFunc != nil {
		return s.spiderFunc(*s.context, s.tool)
	}

	return nil
}

func (s *SkyTracer) UserTraceFunction() {
	ctx := s.getContext()
	span, ctx, err := s.tracer.CreateEntrySpan(ctx, "Entry Function", func() (string, error) {
		if s.request != nil {
			return s.request.Header.Get(propagation.Header), nil
		}
		return "", nil
	})
	if err != nil {
		log.Printf("CreateLocalSpan err:%s", err.Error())
		s.invokeFunction()
		return
	}
	s.setContext(ctx)

	// 注入到logWriter
	subSpan, subCtx, err := s.tracer.CreateLocalSpan(ctx)
	subSpan.SetOperationName("Invoke function")
	s.tool.LogWriter, err = entity.NewSkyTraceWriter(subSpan)

	// 注入到dataWriter
	s.tool.DataWriter.DB.SetTrace(s.tracer, subCtx)
	s.tool.DataWriter.MQ.SetTrace(s.tracer, subCtx)

	span.Log(time.Now(), "INFO", fmt.Sprintf("invoke %s start", s.functionName))
	err = s.invokeFunction()
	defer func() {
		if err != nil {
			span.Error(time.Now(), "ERROR", fmt.Sprintf("invoke %s err: %+v,stack: %s", s.functionName, err, string(debug.Stack())))
		} else {
			span.Log(time.Now(), "INFO", fmt.Sprintf("invoke %s finish", s.functionName))
		}
		span.End()
		s.tool.LogWriter.End()
	}()
}

func (s *SkyTracer) CloseReporter() {
	s.reporter.Close()
}

func (s *SkyTracer) SetFuncName(functionName string) {
	s.functionName = functionName
}

func (s *SkyTracer) getContext() context.Context {
	if s.request != nil {
		return s.request.Context()
	} else {
		return s.context.GetContext()
	}
}

func (s *SkyTracer) setContext(ctx context.Context) {
	if s.request != nil {
		s.request.WithContext(ctx)
	}
	if s.context != nil {
		s.context.SetContext(ctx)
	}
}
