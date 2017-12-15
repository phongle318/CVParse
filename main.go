package main

import (
	"io/ioutil"
	"os"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/gocarina/gocsv"
	"github.com/k3a/html2text"
	"github.com/sajari/docconv"
)

type Client struct { // Our example struct, you can use "-" to ignore a field
	Name  string `csv:"name"`
	Tel   string `csv:"phone"`
	Email string `csv:"email"`
}

func main() {
	//Get folder file
	files, err := ioutil.ReadDir("./CV")
	if err != nil {
		log.Fatal(err)
	}
	clientsFile, err := os.OpenFile("./CV/res/result.csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer clientsFile.Close()
	clients := []*Client{}
	var input string
	for _, f := range files {
		log.Info(f.Name())
		matched, err := regexp.MatchString(".html.*", f.Name())
		if err != nil {
			log.Fatal(err)
		}
		if matched == true {
			file, err := os.Open("./CV/" + f.Name())
			if err != nil {
				log.Error(err)
			}
			defer file.Close()
			b, err := ioutil.ReadAll(file)
			plain := html2text.HTML2Text(string(b[:]))
			input = plain
		} else {
			res, err := docconv.ConvertPath("./CV/" + f.Name())
			if err != nil {
				log.Fatal(err)
			}
			input = res.Body
			log.Infof("res.Body %s", res.Body)
		}
		// Pre Processing
		re := regexp.MustCompile(`\r?\n`)
		inputReplace := re.ReplaceAllString(input, " ")
		log.Infof("inputReplace %s", inputReplace)
		if inputReplace != "" {
			//Start Sending Request
			cont := ContentRequest{}
			cont.Content = inputReplace
			cont.Language = "vi"

			content, err := SendTextForRecognize(cont, "cv")
			if err != nil {
				log.Error(err)
			}
			log.Infof("Content --- > %+v", content)
			if len(content.PersonName) > 0 {
				var client Client
				for i := 0; i < len(content.PersonName); i++ {
					matched, err := regexp.MatchString("Hồ Sơ.*", content.PersonName[i].RealValue)
					if err != nil {
						log.Error(err)
					}
					if matched == false {
						client.Name = content.PersonName[i].RealValue
						break
					}
				}
				if len(content.Email) > 0 {
					client.Email = content.Email[0].RealValue
				}
				if len(content.PhoneNumber) > 0 {
					for i := 0; i < len(content.PhoneNumber); i++ {
						if len(content.PhoneNumber[i].RealValue) > 7 {
							client.Tel = content.PhoneNumber[i].RealValue
						}
					}
				}
				clients = append(clients, &client)
			}
		}
	}
	err = gocsv.MarshalFile(&clients, clientsFile) // Use this to save the CSV back to the file
	if err != nil {
		log.Error(err)
	}
}
