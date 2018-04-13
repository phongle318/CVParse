package main

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/k3a/html2text"
	"github.com/sajari/docconv"
)

func parseCV(w http.ResponseWriter, r *http.Request) {
	var client Client

	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("attachment")
	if err != nil {
		log.Errorf("Error happen FormFile attachment : %s", err)
		ResponseError(w, err)
		return
	}
	defer file.Close()
	client.FileName = handler.Filename
	CreateDirIfNotExist(RootFolder)
	folderGen := InputFolder + time.Now().Format("20060102") + "/"
	CreateDirIfNotExist(folderGen)
	f, err := os.OpenFile(folderGen+handler.Filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Errorf("Error happen Open file : %s", err)
		ResponseError(w, err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	var input string

	getFile, err := os.Open(folderGen + handler.Filename)
	if err != nil {
		log.Errorf("Error happen in Open folderGen File : %s", err)
		ResponseError(w, err)
		return
	}
	defer getFile.Close()
	htmlFile, err := regexp.MatchString(".html.*", handler.Filename)
	if err != nil {
		log.Errorf("Error happen MatchString html: %s", err)
		ResponseError(w, err)
		return
	}
	if htmlFile == true {
		file, err := os.Open(folderGen + handler.Filename)
		if err != nil {
			log.Errorf("Error happen in Open folderGen File : %s", err)
			ResponseError(w, err)
			return
		}
		defer file.Close()
		b, err := ioutil.ReadAll(file)
		plain := html2text.HTML2Text(string(b[:]))
		input = plain
		log.Infof("input %s", input)
	} else {
		pdfMatched, err := regexp.MatchString(".pdf.*", handler.Filename)
		if err != nil {
			log.Errorf("Error happen MatchString pdf: %s", err)
			ResponseError(w, err)
			return
		}
		if pdfMatched == true {
			out, err := exec.Command("python3", "/usr/local/bin/pdf2txt.py", "-W0", folderGen+"/"+handler.Filename).Output()
			if err != nil {
				log.Errorf("Error happen exec.Command pdf: %s", err)
				ResponseError(w, errors.New("Cannot processing this file"))
				return
			}
			input = BytesToString(out)
			log.Infof(input)
		} else {
			log.Info(handler.Filename)
			res, err := docconv.ConvertPath(folderGen + handler.Filename)
			if err != nil {
				log.Errorf("Error happen in ConvertPath : %s", err)
				ResponseError(w, errors.New("Cannot processing this file"))
				return
			}
			log.Info(res)
			input = res.Body
		}
	}
	if input == "" {
		log.Error("Cannot processing this file")
		//ResponseError(w, errors.New("Cannot processing this file"))
		client.Message = err.Error()
		ResponseJSON(w, client)
		return
	}


	inputReplace, err := PreProcressing(input)
	if err != nil {
		log.Errorf("Error happen in PreProcressing : %s", err)
		client.Message = err.Error()
		ResponseJSON(w, client)
		return
	}
	if inputReplace != "" {
		//Start Sending Request
		cont := ContentRequest{}
		cont.Content = inputReplace
		cont.Language = "vi"

		content, err := SendTextForRecognize(cont, "cv")
		if err != nil {
			log.Errorf("Error happen in SendTextForRecognize : %s", err)
			client.Message = err.Error()
			//ResponseError(w, err)
			ResponseJSON(w, client)
			return
		}
		name := ""
		if len(content.PersonName) > 0 {
			client.FileName = handler.Filename
			client.Content = "[" + inputReplace + "]"
			// for i := 0; i < len(content.PersonName); i++ {
			// 	name = name + content.PersonName[i].RealValue + "\n"
			// }
			for i := 0; i < len(content.PersonName); i++ {
				if IsRightName(strings.ToLower(content.PersonName[i].RealValue)) {
					name = content.PersonName[i].RealValue
					break
				}
			}
			client.Name = name
			if len(content.Email) > 0 {
				client.Email = content.Email[0].RealValue
			}
			if len(content.PhoneNumber) > 0 {
				client.Tel = content.PhoneNumber[0].Value
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	// w.Write([]byte(fileBase))
	ResponseJSON(w, client)
}
