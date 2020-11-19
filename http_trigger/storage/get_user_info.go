package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type UserInfo struct {
	Name   string
	Age    int
	Sex    string
	IDCard string
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		user := UserInfo{
			Name:   "ryan",
			Age:    24,
			Sex:    "male",
			IDCard: "3606",
		}
		data, err := json.Marshal(user)
		time.Sleep(2 * time.Second)
		if err != nil {
			log.Printf("json marshal err%s \n", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write(data)
	case http.MethodPost:
		time.Sleep(5 * time.Second)
		var slice []int
		fmt.Println(slice[1])
	}
}
