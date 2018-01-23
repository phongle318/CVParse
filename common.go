package main

import (
	"encoding/json"
	"net/http"
	"os"

	log "github.com/Sirupsen/logrus"
)

func ResponseError(w http.ResponseWriter, err error) {
	log.Error("Error API request: ", err)
	w.WriteHeader(http.StatusBadRequest)
	ResponseJSON(w, map[string]interface{}{"error": err.Error()})
}

func ResponseJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	respBody, err := json.Marshal(v)
	if err != nil {
		log.Error("Error in responseJSON: ", err.Error())
		respBody, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(respBody)
	} else {
		w.Write(respBody)
	}
}

func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

func IsRightName(PersonName string) bool {
	switch PersonName {
	case "hồ sơ":
		log.Info("case False")
		return false
	case "lý lịch":
		log.Info("case False")
		return false
	case "công ty":
		log.Info("case False")
		return false
	case "công ti":
		log.Info("case False")
		return false
	case "ho chi minh":
		log.Info("case False")
		return false
	case "hồ chí minh":
		log.Info("case False")
		return false
	case "duy tan":
		log.Info("case False")
		return false
	case "ha noi":
		log.Info("case False")
		return false
	case "hà nội":
		log.Info("case False")
		return false
	case "tran nao":
		log.Info("case False")
		return false
	default:
		log.Info("case true")
		return true

	}
	return false
}
