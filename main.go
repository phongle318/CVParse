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
	"unicode/utf8"
	"errors"

	"math"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/gocarina/gocsv"
	"github.com/gorilla/mux"
	"github.com/k3a/html2text"
	"github.com/mholt/archiver"
	"github.com/sajari/docconv"
)

const (
	RootFolder   string = "./CV/"
	InputFolder  string = "./CV/input_folder"
	OutputFolder string = "./CV/output_folder"
	ResultFolder string = "./CV/result"
)

type Client struct { // Our example struct, you can use "-" to ignore a field
	FileName string `csv:"filename" json:"filename"`
	Tel      string `csv:"phone" json:"phone" default:" "`
	Email    string `csv:"email" json:"email" default:" "`
	Name     string `csv:"name" json:"name" default:" "`
	Content  string `csv:"content" json:"-"`
	Message  string `csv:"message" json:"message" default:"success"`
}

func main() {
	log.Info("Start main")
	r := mux.NewRouter()

	r.HandleFunc("/parsezip", parseZip).Methods("POST")
	r.HandleFunc("/parsecv", parseCV).Methods("POST")
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

func parseZip(w http.ResponseWriter, r *http.Request) {
	// createDirIfNotExist("/CV")
	r.ParseMultipartForm(32 << 20)
	log.Info("start")
	log.Info(r)
	file, handler, err := r.FormFile("attachment")
	if err != nil {
		log.Errorf("Error happen FormFile attachment : %s", err)
		return
	}
	defer file.Close()
	CreateDirIfNotExist(RootFolder)
	folderGen := InputFolder + time.Now().Format("20060102150405") + "/"
	CreateDirIfNotExist(folderGen)
	f, err := os.OpenFile(folderGen+handler.Filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Errorf("Error happen Open file : %s", err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	extractOutput := OutputFolder + time.Now().Format("20060102150405") + "/"
	log.Info("handler.Filename")
	log.Info(handler.Filename)
	err = archiver.Zip.Open(folderGen+handler.Filename, extractOutput)
	if err != nil {
		log.Errorf("Error happen Extract Zip : %s", err)
		ResponseError(w, err)
		return
	}
	//Get folder file
	log.Info("ReadDir")
	files, err := ioutil.ReadDir(extractOutput)
	if err != nil {
		log.Errorf("Error happen ReadDir : %s", err)
		ResponseError(w, err)
		return
	}

	CreateDirIfNotExist(ResultFolder)
	clientFileName := "/result_" + time.Now().Format("20060102150405") + ".csv"
	log.Info("OpenFile")
	clientsFile, err := os.OpenFile(ResultFolder+clientFileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Errorf("Error happen OpenFile : %s", err)
		ResponseError(w, err)
		return
	}
	defer clientsFile.Close()
	clients := []*Client{}
	var input string
	for f := 0; f < len(files); f++ {
		log.Info(files[f].Name())
		log.Info("MatchString")
		matched, err := regexp.MatchString(".html.*", files[f].Name())
		if err != nil {
			log.Fatalf("Fatal happen MatchString html: %s", err)
			return
		}
		if matched == true {
			file, err := os.Open(extractOutput + files[f].Name())
			if err != nil {
				log.Errorf("Error happen in Open extractOutput File : %s", err)
				return
			}
			defer file.Close()
			b, err := ioutil.ReadAll(file)
			plain := html2text.HTML2Text(string(b[:]))
			input = plain
			log.Infof("input %s", input)
		} else {
			pdfMatched, err := regexp.MatchString(".pdf.*", files[f].Name())
			if err != nil {
				log.Fatalf("Fatal happen MatchString pdf: %s", err)
				return
			}
			if pdfMatched == true {
				out, err := exec.Command("python3", "/usr/local/bin/pdf2txt.py", "-W0", extractOutput+"/"+files[f].Name()).Output()
				if err != nil {
					log.Fatal(err)
				}
				input = BytesToString(out)
				log.Infof(input)
			} else {
				res, err := docconv.ConvertPath(extractOutput + files[f].Name())
				if err != nil {
					log.Errorf("Error happen in ConvertPath : %s", err)
					return
				}
				input = res.Body
			}
		}
		if input == "" {
			log.Errorf("Error happen in input : %s", err)
			continue
		}

		inputReplace, err := PreProcressing(input)
		if err != nil {
			log.Errorf("Error happen in PreProcressing : %s", err)
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
				return
			}
			name := ""
			if len(content.PersonName) > 0 {
				var client Client
				client.FileName = files[f].Name()
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
				clients = append(clients, &client)
			}
		}
	}

	err = gocsv.MarshalFile(&clients, clientsFile) // Use this to save the CSV back to the file
	if err != nil {
		log.Errorf("Error happen in MarshalFile : %s", err)
	}
	// w.Header().Set("Content-Type", "application/json")
	// w.Write([]byte(fileBase))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Set("Content-Disposition", "attachment; filename='attachment.zip'")

	io.Copy(w, clientsFile)
	http.ServeFile(w, r, clientsFile.Name())
}

func PreProcressing(input string) (result string, err error) {
	log.Info("PreProcressing")
	log.Info(input)
	re1 := regexp.MustCompile(`(\\a)|((<\s*br\s*\/?>)|[\t\n\r])`)
	re2 := regexp.MustCompile(`([\s]|&nbsp;){2,}`)
	re3 := regexp.MustCompile(`(\s;){2,}`)
	//inputReplace := re.ReplaceAllString(input, " ") //replace with space
	inputReplace := re1.ReplaceAllString(input, " ; ") //replace with ;
	inputReplace = html2text.HTML2Text(inputReplace)
	inputReplace = re2.ReplaceAllString(inputReplace, " ")   //replace with space
	inputReplace = re3.ReplaceAllString(inputReplace, " ; ") //replace multi ; with ;
	inputReplace = strings.Join(strings.Fields(inputReplace), " ")
	log.Infof("Ket Qua %s", inputReplace)
	numberOfText := math.Min(float64(utf8.RuneCountInString(inputReplace)), 2048)

	log.Info(len(inputReplace))
	log.Info(numberOfText)

	inputReplace = string([]rune(inputReplace)[:int(numberOfText)])

	words := strings.Fields(inputReplace)
	for i, word := range words {
		log.Info(i, " => ", word)
		log.Info(len(word))
		if len(word) > 60 {
			return "", errors.New("Cannot processing this file")
		}
	}

	log.Info(inputReplace)
	return inputReplace, nil
}

func BytesToString(data []byte) string {
	return string(data[:])
}
