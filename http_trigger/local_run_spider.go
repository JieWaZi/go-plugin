package main

import (
	"github.com/JieWazi/goplugin/http_trigger/entity"
	"github.com/JieWazi/goplugin/http_trigger/global"
	"github.com/JieWazi/goplugin/http_trigger/middleware"
	"github.com/JieWazi/goplugin/http_trigger/utils"
	"log"
	"os"
	"plugin"
)

func main() {
	executeSpider("spider.go", "Spider")
}

func executeSpider(fileName, functionName string) {
	spiderFunc, err := load(fileName, functionName)
	if err != nil {
		panic(err)
	}
	tracer := global.Conf.SkyTrace
	defer tracer.CloseReporter()
	tracer.SetFuncName(functionName)
	tracer.WithSpiderFunc(spiderFunc, &middleware.JsonContext{},
		&middleware.FunctionTool{
			DataWriter: middleware.DataWriter{
				DB: global.Conf.MySQL,
				MQ: global.Conf.Kafka,
			},
			LogWriter: entity.SkyTraceWriter{},
		})
	tracer.UserTraceFunction()
}

func load(fileName, functionName string) (middleware.SpiderFunc, error) {
	loader, err := utils.NewLoader(fileName, functionName)
	if err != nil {
		log.Printf("NewLoader  err:%s \n", err.Error())
		return nil, err
	}

	err = loader.Compile()
	if err != nil {
		if err == os.ErrNotExist {
			log.Printf("get file by function path is not exist  err:%s \n", err.Error())
		}
		return nil, err
	}

	sym, err := loader.LoadPlugin()
	if err != nil {
		return nil, err
	}
	return loadP(sym), nil
}

func loadP(sym plugin.Symbol) middleware.SpiderFunc {
	switch h := sym.(type) {
	case *middleware.SpiderFunc:
		return *h
	case func(middleware.Context, *middleware.FunctionTool) error:
		return h
	default:
		log.Println("not spider plugin")
		return nil
	}
}
