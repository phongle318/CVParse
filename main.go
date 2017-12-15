package main

import (
	"github.com/sajari/docconv"
	"regexp"
	"io/ioutil"
	log "github.com/Sirupsen/logrus"
	"os"
	"github.com/gocarina/gocsv"
)

type Client struct { // Our example struct, you can use "-" to ignore a field
	Name    string `csv:"client_name"`
	Tel     string `csv:"client_age"`
	Email 	string `csv:"email"`
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

    for _, f := range files {
		// log.Info(f.Name())
		res, err := docconv.ConvertPath("./CV/"+f.Name())
		if err != nil {
			log.Fatal(err)
		}

		// Pre Processing
		re := regexp.MustCompile(`\r?\n`)
		input := re.ReplaceAllString(res.Body, " ")
		log.Infof("res.Body %s", res.Body)
		log.Infof("input %s", input)
		if input != ""{
			//Start Sending Request
			cont := ContentRequest{}
			cont.Content = input
			cont.Language = "vi"
			
			content, err := SendTextForRecognize(cont, "personname")
			if err != nil {
				log.Error(err)
			}
			log.Infof("Content --- > %+v", content)
			if len(content) > 0{
				clients = append(clients, &Client{Name: content[0].Value, Tel: "0",Email:"0"}) 
			}
		}
	}
	err = gocsv.MarshalFile(&clients, clientsFile) // Use this to save the CSV back to the file
	if err != nil {
		log.Error(err)
	}
}