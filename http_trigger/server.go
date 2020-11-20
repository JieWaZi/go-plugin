package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JieWazi/goplugin/http_trigger/entity"
	"github.com/JieWazi/goplugin/http_trigger/global"
	"github.com/JieWazi/goplugin/http_trigger/middleware"
	"github.com/JieWazi/goplugin/http_trigger/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"plugin"
)

func main() {
	tracer := global.Conf.SkyTrace
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
			var context middleware.JsonContext
			body, err := ioutil.ReadAll(r.Body)
			if checkStatusInternalServerError(w, err, "read body err") {
				return
			}
			err = json.Unmarshal(body, &context)
			if checkStatusInternalServerError(w, err, "unmarshal body err") {
				return
			}
			tracer.WithSpiderFunc(spiderFunc, &context, &middleware.FunctionTool{
				DataWriter: middleware.DataWriter{
					DB: global.Conf.MySQL,
					MQ: global.Conf.Kafka,
				},
				LogWriter: entity.SkyTraceWriter{},
			})
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

var httpFunc middleware.HttpFunc
var spiderFunc middleware.SpiderFunc
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

func loadHTTPPlugin(sym plugin.Symbol) middleware.HttpFunc {
	switch h := sym.(type) {
	case *http.Handler:
		return (*h).ServeHTTP
	case *http.HandlerFunc:
		return middleware.HttpFunc(*h)
	case *middleware.HttpFunc:
		return *h
	case func(http.ResponseWriter, *http.Request):
		return h
	case func(context.Context, http.ResponseWriter, *http.Request):
		return func(w http.ResponseWriter, r *http.Request) {
			c := context.Background()
			h(c, w, r)
		}
	default:
		log.Println("not http plugin")
		return nil
	}
}

func loadSpiderPlugin(sym plugin.Symbol) middleware.SpiderFunc {
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
			log.Println(err)
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
