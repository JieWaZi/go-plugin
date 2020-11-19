package utils

import (
	"context"
	"fmt"
	"github.com/JieWazi/goplugin/http_trigger/func_plugin"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/propagation"
	"github.com/SkyAPM/go2sky/reporter"
	"github.com/sirupsen/logrus"
	"net/http"
	"runtime/debug"
	"time"
)

type SkyTracer struct {
	endpoint     string
	serviceName  string
	functionName string
	tracer       *go2sky.Tracer
	reporter     go2sky.Reporter

	response http.ResponseWriter
	request  *http.Request
	httpFunc func_plugin.HttpFunc

	context    *func_plugin.Context
	tool       *func_plugin.FunctionTool
	spiderFunc func_plugin.SpiderFunc
}

func InitTracer(endpoint, serviceName string) (*SkyTracer, error) {
	var (
		err error
	)
	s := &SkyTracer{
		endpoint:    endpoint,
		serviceName: serviceName,
	}
	s.reporter, err = reporter.NewGRPCReporter(endpoint)
	if err != nil {
		logrus.Errorf("new reporter error:%v", err.Error())
		return nil, err
	}
	s.tracer, err = go2sky.NewTracer(serviceName, go2sky.WithReporter(s.reporter))
	if err != nil {
		logrus.Errorf("NewTracer error:%v", err.Error())
		return nil, err
	}
	return s, nil
}

func (s *SkyTracer) WithSpiderFunc(spiderFunc func_plugin.SpiderFunc, jctx *func_plugin.JsonContext, tool *func_plugin.FunctionTool) {
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
	s.context = func_plugin.NewContext(ctx, ctxMap)
}

func (s *SkyTracer) WithHttpFunc(httpFunc func_plugin.HttpFunc, resp http.ResponseWriter, req *http.Request) {
	s.httpFunc = httpFunc
	s.request = req
	s.response = resp
}

func (s *SkyTracer) invokeFunction() error {
	if s.httpFunc != nil {
		s.httpFunc(s.response, s.request)
	} else if s.spiderFunc != nil {
		s.spiderFunc(*s.context, s.tool)
	}

	return nil
}

func (s *SkyTracer) UserTraceFunction() {
	ctx := s.getContext()
	span, ctx, err := s.tracer.CreateEntrySpan(ctx, "Invoke Function", func() (string, error) {
		if s.request != nil {
			return s.request.Header.Get(propagation.Header), nil
		}
		return "", nil
	})
	if err != nil {
		logrus.Errorf("CreateLocalSpan err:%s", err.Error())
		s.invokeFunction()
		return
	}
	s.setContext(ctx)
	// 注入到logWriter
	s.tool.LogWriter = func_plugin.NewSkyTraceWriter(span)

	// 注入到dataWriter
	s.tool.DataWriter.DB.SetTrace(s.tracer, ctx)
	s.tool.DataWriter.MQ.SetTrace(s.tracer, ctx)

	span.Log(time.Now(), "INFO", fmt.Sprintf("invoke %s start", s.functionName))
	s.invokeFunction()
	defer func() {
		if err := recover(); err != nil {
			span.Error(time.Now(), "ERROR", fmt.Sprintf("invoke %s err: %+v, stack:\n%s", s.functionName, err, string(debug.Stack())))
		} else {
			span.Log(time.Now(), "INFO", fmt.Sprintf("invoke %s finish", s.functionName))
		}
		span.End()
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
