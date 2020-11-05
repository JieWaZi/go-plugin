package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"learn_go/go_plugin/http_trigger/utils"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/specialize", specializeHandler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if userFunc == nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("init function code is failed, please specialize first"))
			return
		}
		userFunc(w, r)
	})

	log.Println("listening on 9100 ...")
	http.ListenAndServe(":9100", nil)
}

type Function struct {
	FunctionName string `json:"functionName"`
	FileName     string `json:"fileName"`
}

var userFunc http.HandlerFunc

func specializeHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Printf("specializeHandler read request body err:%s \n", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
			defer r.Body.Close()

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
			userFunc, err = loadPlugin(loader)
			if err != nil {
				log.Printf("loadPlugin err:%s \n", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
	}
}

func loadPlugin(loader *utils.Loader) (http.HandlerFunc, error) {
	sym, err := loader.LoadPlugin()
	if err != nil {
		return nil, err
	}
	switch h := sym.(type) {
	case *http.Handler:
		return (*h).ServeHTTP, nil
	case *http.HandlerFunc:
		return *h, nil
	case func(http.ResponseWriter, *http.Request):
		return h, nil
	case func(context.Context, http.ResponseWriter, *http.Request):
		return func(w http.ResponseWriter, r *http.Request) {
			c := context.Background()
			h(c, w, r)
		}, nil
	default:
		panic("Entry point not found: bad type")
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
