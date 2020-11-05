package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type UserInfo struct {
	Name   string
	Age    int
	Sex    string
	IDCard string
}

func GetUserInfos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		user := UserInfo{
			Name:   "ryan",
			Age:    24,
			Sex:    "male",
			IDCard: "3606",
		}
		data, err := json.Marshal([]UserInfo{user})
		if err != nil {
			log.Printf("json marshal err%s \n", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write(data)
	}
}
