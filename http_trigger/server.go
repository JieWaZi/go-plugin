package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JieWazi/goplugin/http_trigger/func_plugin"
	"github.com/JieWazi/goplugin/http_trigger/utils"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"plugin"
)

func main() {
	endpoint := os.Getenv("SKY_TRACE_ENDPOINT")
	logrus.Infof("endpoint:%s", endpoint)
	serviceName := os.Getenv("SERVICE_NAME")
	logrus.Infof("serviceName:%s", serviceName)
	tracer, err := utils.InitTracer(endpoint, serviceName)
	if err != nil {
		panic(err)
	}
	defer tracer.CloseReporter()
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/specialize", specializeHandler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if httpFunc == nil && spiderFunc == nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("init function code is failed, please specialize first"))
			return
		}
		tracer.SetFuncName(funcName)
		if httpFunc != nil {
			tracer.WithHttpFunc(httpFunc, w, r)
		} else if spiderFunc != nil {
			defer r.Body.Close()
			var context func_plugin.JsonContext
			body, err := ioutil.ReadAll(r.Body)
			if checkStatusInternalServerError(w, err, "read body err") {
				return
			}
			err = json.Unmarshal(body, &context)
			if checkStatusInternalServerError(w, err, "unmarshal body err") {
				return
			}
			tool, err := func_plugin.InitFunctionTool()
			if checkStatusInternalServerError(w, err, "init function tool err") {
				return
			}
			tracer.WithSpiderFunc(spiderFunc, &context, tool)
		}
		tracer.UserTraceFunction()
	})

	log.Println("listening on 9100 ...")
	http.ListenAndServe(":9100", nil)
}

type Function struct {
	FunctionName string `json:"functionName"`
	FileName     string `json:"fileName"`
}

var httpFunc func_plugin.HttpFunc
var spiderFunc func_plugin.SpiderFunc
var funcName string

func specializeHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Printf("specializeHandler read request body err:%s \n", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}

			var fnc Function
			err = json.Unmarshal(data, &fnc)
			if err != nil {
				log.Printf("unmarshal function err:%s \n", err.Error())
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			loader, err := utils.NewLoader(fnc.FileName, fnc.FunctionName)
			if err != nil {
				log.Printf("NewLoader  err:%s \n", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			err = loader.Compile()
			if err != nil {
				if err == os.ErrNotExist {
					log.Printf("get file by function path is not exist  err:%s \n", err.Error())
					w.WriteHeader(http.StatusBadRequest)
					return
				} else {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			err = loadPlugin(loader)
			if checkStatusInternalServerError(w, err, "load plugin err") {
				return
			}
			funcName = fnc.FunctionName
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}
}

func loadPlugin(loader *utils.Loader) error {
	sym, err := loader.LoadPlugin()
	if err != nil {
		return err
	}
	httpFunc = loadHTTPPlugin(sym)
	spiderFunc = loadSpiderPlugin(sym)
	if httpFunc == nil && spiderFunc == nil {
		return nil
	}
	return nil
}

func loadHTTPPlugin(sym plugin.Symbol) func_plugin.HttpFunc {
	switch h := sym.(type) {
	case *http.Handler:
		return (*h).ServeHTTP
	case *http.HandlerFunc:
		return func_plugin.HttpFunc(*h)
	case *func_plugin.HttpFunc:
		return *h
	case func(http.ResponseWriter, *http.Request):
		return h
	case func(context.Context, http.ResponseWriter, *http.Request):
		return func(w http.ResponseWriter, r *http.Request) {
			c := context.Background()
			h(c, w, r)
		}
	default:
		logrus.Infof("not http plugin")
		return nil
	}
}

func loadSpiderPlugin(sym plugin.Symbol) func_plugin.SpiderFunc {
	switch h := sym.(type) {
	case *func_plugin.SpiderFunc:
		return *h
	case func(func_plugin.Context, *func_plugin.FunctionTool):
		return h
	default:
		logrus.Infof("not spider plugin")
		return nil
	}
}

func upload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(http.StatusMethodNotAllowed)
	case http.MethodPost:
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			log.Printf("ParseMultipartForm  err:%s \n", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		remoteFD, file, err := r.FormFile("file")
		if err != nil {
			fmt.Println(err)
			w.Write([]byte("upload failed."))
			return
		}
		path := os.Getenv("PWD")
		localPath := fmt.Sprintf("%s/storage/%s", path, file.Filename)
		localFD, err := os.OpenFile(localPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		io.Copy(localFD, remoteFD)
		w.Write([]byte("upload finish"))

	}
}

func checkStatusInternalServerError(w http.ResponseWriter, err error, errInfo string) bool {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("%s:%s", errInfo, err.Error())))
		return true
	}
	return false
}
