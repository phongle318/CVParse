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
	"encoding/json"
		
	// "syscall"
	"os/exec"

	log "github.com/Sirupsen/logrus"
	"github.com/gocarina/gocsv"
	"github.com/gorilla/mux"
	"github.com/k3a/html2text"
	"github.com/mholt/archiver"
	"github.com/sajari/docconv"
)
const (
	RootFolder string = "./CV/"
	InputFolder string = "./CV/input_folder"
	OutputFolder string = "./CV/output_folder"
	ResultFolder string = "./CV/result"
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
	// createDirIfNotExist("/CV")
	r.ParseMultipartForm(32 << 20)
	file, handler, err := r.FormFile("attachment")
	if err != nil {
		log.Info(err)
		return
	}
	log.Info(file)
	log.Info("file")
	defer file.Close()
	CreateDirIfNotExist(RootFolder)
	folderGen := InputFolder+time.Now().Format("20060102150405") + "/"
	CreateDirIfNotExist(folderGen)
	f, err := os.OpenFile(folderGen+handler.Filename, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Errorf("Error happen Open file : %s", err)
		return
	}
	defer f.Close()
	io.Copy(f, file)

	extractOutput := OutputFolder + time.Now().Format("20060102150405") + "/"
	log.Info("handler.Filenam")
	log.Info(handler.Filename)
	err = archiver.Zip.Open(folderGen+handler.Filename, extractOutput)
	if err != nil {
		log.Errorf("Error happen Extract Zip : %s", err)
		ResponseError(w, err)
		return
	}
	//Get folder file
	files, err := ioutil.ReadDir(extractOutput)
	if err != nil {
		log.Errorf("Error happen ReadDir : %s", err)
		ResponseError(w, err)
		return
	}

	CreateDirIfNotExist(ResultFolder)
	clientFileName := "/result_"+time.Now().Format("20060102150405")+".csv"

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
			} else {
				res, err := docconv.ConvertPath(extractOutput + files[f].Name())
				if err != nil {
					log.Errorf("Error happen in ConvertPath : %s", err)
					return
				}
				input = res.Body
			}
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
		// inputReplace = string([]rune(inputReplace)[:1024])

		if inputReplace != "" {
			//Start Sending Request
			cont := ContentRequest{}
			cont.Content = inputReplace
			cont.Language = "vi"

			content, err := SendTextForRecognize(cont, "cv")
			if err != nil {
				log.Errorf("Error happen in SendTextForRecognize : %s", err)
				continue
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
		log.Errorf("Error happen in MarshalFile : %s", err)
	}
	// w.Header().Set("Content-Type", "application/json")
	// w.Write([]byte(fileBase))
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Header().Set("Content-Disposition", "attachment; filename='attachment.zip'")

	io.Copy(w, clientsFile)

	http.ServeFile(w, r, clientsFile.Name())
}

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
	case "hồ sơ" :
		log.Info("case False")
		return false
	case "lý lịch" :
		log.Info("case False")
		return false
	case "công ty" :
		log.Info("case False")
		return false
	case "công ti" :
		log.Info("case False")
		return false
	case "ho chi minh" :
		log.Info("case False")
		return false
	case "hồ chí minh" :
		log.Info("case False")
		return false
	case "duy tan" :
		log.Info("case False")
		return false
	case "ha noi" :
		log.Info("case False")
		return false
	case "hà nội" :
		log.Info("case False")
		return false
	default:
		log.Info("case true")
		return true

	}
	return false
}

func BytesToString(data []byte) string {
	return string(data[:])
}