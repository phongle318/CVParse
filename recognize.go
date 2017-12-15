package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

var regEndpoint = "http://104.197.234.1:9009/"

type ContentRequest struct {
	Content  string `json:"content"`
	Language string `json:"language"`
}

type Result struct {
	PersonName  []Item `json:"person_name"`
	Email       []Item `json:"email"`
	PhoneNumber []Item `json:"phone_number"`
}

type Item struct {
	Value     string `json:"value"`
	End       int    `json:"end"`
	Start     int    `json:"start"`
	Type      string `json:"type"`
	RealValue string `json:"real_value"`
}

func SendTextForRecognize(content ContentRequest, uri string) (
	result Result, err error) {
	log.Info("Start Send Text")
	URI := regEndpoint + uri

	jsonParam, err := json.Marshal(content)
	if err != nil {
		return result, errors.Wrapf(err, "Marshal request failed: input=%+v\n", jsonParam)
	}

	req, err := http.NewRequest("POST", URI, bytes.NewReader(jsonParam))
	if err != nil {
		return result, errors.Wrapf(err, "NewRequest failed: %s %s\n", "POST", URI)
	}

	req.Header.Set("Content-Type", "application/json")
	tr := &http.Transport{}
	httpClient := &http.Client{
		Timeout:   100 * time.Second,
		Transport: tr,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return result, errors.Wrapf(err, "Do failed")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, errors.Wrapf(err, "ReadAll failed: body=%+v\n", resp.Body)
	}
	defer resp.Body.Close()

	if err = json.Unmarshal(body, &result); err != nil {
		return result, errors.Wrapf(err, "Decode failed: body=%s\n", string(body))
	}

	log.Infof("Decode > %+v", result)
	log.Info("End Send Text")
	return result, nil
}
