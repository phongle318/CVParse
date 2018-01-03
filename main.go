package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/gocarina/gocsv"
	"github.com/gorilla/mux"
	"github.com/k3a/html2text"
	"github.com/mholt/archiver"
	"github.com/sajari/docconv"
)

type Client struct { // Our example struct, you can use "-" to ignore a field
	FileName string `csv:"filename"`
	Tel      string `csv:"phone"`
	Email    string `csv:"email"`
	Name     string `csv:"name"`
	Content  string `csv:"content"`
}

func main() {
	log.Info("Start main")
	r := mux.NewRouter()

	r.HandleFunc("/parseCV", parseCV).Methods("POST")
	r.HandleFunc("/", viewCVForm)

	log.Info("Listening at port:", "8050")

	//start server
	log.Fatal(http.ListenAndServe(":8050", r))
}

func viewCVForm(w http.ResponseWriter, r *http.Request) {
	// w.Header().Set("Access-Control-Allow-Origin", "*")
	t, _ := template.New("cv.html").Delims("{[{", "}]}").ParseFiles("cv.html")
	t.Execute(w, nil)
}

func parseCV(w http.ResponseWriter, r *http.Request) {
	log.Info("Parse CV")
	log.Info(r)

	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("attachment")
	if err != nil {
		log.Info(err)
		return
	}
	defer file.Close()
	log.Infof("%+v", handler.Header)
	log.Infof("%+v", w)
	f, err := os.OpenFile("./CV/test/"+handler.Filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Info(err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	extractOutput := "./CV/output_folder" + time.Now().Format("20060102150405") + "/"

	err = archiver.Zip.Open("./CV/test/"+handler.Filename, extractOutput)
	if err != nil {
		log.Info(err)
		return
	}
	//Get folder file
	files, err := ioutil.ReadDir(extractOutput)
	if err != nil {
		log.Fatal(err)
	}
	clientsFile, err := os.OpenFile("./CV/res/result_"+time.Now().Format("20060102150405")+".csv", os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer clientsFile.Close()
	clients := []*Client{}
	var input string
	for f := 0; f < len(files); f++ {
		log.Info(files[f].Name())
		matched, err := regexp.MatchString(".html.*", files[f].Name())
		if err != nil {
			log.Fatal(err)
		}
		if matched == true {
			file, err := os.Open(extractOutput + files[f].Name())
			if err != nil {
				log.Error(err)
			}
			defer file.Close()
			b, err := ioutil.ReadAll(file)
			plain := html2text.HTML2Text(string(b[:]))
			input = plain
			log.Infof("input %s", input)
		} else {
			doc, err := regexp.MatchString(".doc.*", files[f].Name())
			if err != nil {
				log.Fatal(err)
			}
			if doc == true {
				log.Infof("Cannot handle doc : %s", files[f].Name())
				continue
			}
			res, err := docconv.ConvertPath(extractOutput + files[f].Name())
			if err != nil {
				log.Error(err)
				return
			}
			input = res.Body
		}
		// Pre Processing
		// input = strings.Join(strings.Fields(input), " ")
		// log.Infof("standard input %s",input)
		// re := regexp.MustCompile(`([\t\n\r]|(&nbsp;)){3,}`)
		re := regexp.MustCompile(`([\s]|&nbsp;){3,}|((<\s*br\s*\/?>)|[\t\n\r])`)
		//inputReplace := re.ReplaceAllString(input, " ") //replace with space
		inputReplace := re.ReplaceAllString(input, " ; ") //replace with ;
		inputReplace = strings.Join(strings.Fields(inputReplace), " ")
		log.Infof("Ket Qua %s", inputReplace)
		if inputReplace != "" {
			//Start Sending Request
			cont := ContentRequest{}
			cont.Content = inputReplace
			cont.Language = "vi"

			content, err := SendTextForRecognize(cont, "cv")
			if err != nil {
				log.Error(err)
			}
			name := ""
			if len(content.PersonName) > 0 {
				var client Client
				client.FileName = files[f].Name()
				client.Content = "[" + inputReplace + "]"
				for i := 0; i < len(content.PersonName); i++ {
					name = name + content.PersonName[i].RealValue + "\n"
				}
				client.Name = name
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
	// fileName := clientsFile.Name()
	// fileBase := filepath.Base(fileName)
	err = gocsv.MarshalFile(&clients, clientsFile) // Use this to save the CSV back to the file
	if err != nil {
		log.Error(err)
	}
	// w.Header().Set("Content-Type", "application/json")
	// w.Write([]byte(fileBase))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Set("Content-Disposition", "attachment; filename='attachment.zip'")

	io.Copy(w, clientsFile)

	http.ServeFile(w, r, clientsFile.Name())
}
