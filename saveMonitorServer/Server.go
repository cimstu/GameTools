package main

import (
    "io"
    "net/http"
    "os"
    "fmt"
	"io/ioutil"
	"strings"
	"encoding/json"
)

const port = "80"

func defaultHandler(w http.ResponseWriter, r* http.Request){
    fmt.Println("welcome!")

}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("upload begin")
    switch r.Method {
    //POST takes the uploaded file(s) and saves it to disk.
    case "POST":
	fmt.Println("upload Post come in")
        //parse the multipart form in the request
        err := r.ParseMultipartForm(100000)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        //get a ref to the parsed multipart form
        m := r.MultipartForm

        //get the *fileheaders
        files := m.File["uploadfile"]
        for i, _ := range files {
            //for each fileheader, get a handle to the actual file
            file, err := files[i].Open()
            defer file.Close()
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            //create destination file making sure the path is writeable.
            dst, err := os.Create("./upload/" + files[i].Filename)
            defer dst.Close()
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            //copy the uploaded file to the destination file
            if _, err := io.Copy(dst, file); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }

        }

    default:
        w.WriteHeader(http.StatusMethodNotAllowed)
    }
}

func getlistHandler(w http.ResponseWriter, r* http.Request) {
	os.MkdirAll("./upload", 0777)

	uploadDir, err := ioutil.ReadDir("./upload")
	if err != nil {
		http.Error(w, "no upload folder", http.StatusInternalServerError)
		return
	}

	var fileList []string

	for _, f := range uploadDir {
		if f == nil ||
		f.IsDir() ||
		!strings.HasSuffix(f.Name(), ".up") {
			continue
		}

		fileList = append(fileList, f.Name())
	}

	b, err := json.Marshal(fileList)
	if err != nil {
		http.Error(w, "json marshell failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application-json")
	w.Write(b)
}

func main() {
    http.HandleFunc("/upload", uploadHandler)

    http.HandleFunc("/getlist", getlistHandler)

    //static file handler.
    http.Handle("/staticfile/", http.StripPrefix("/staticfile/", http.FileServer(http.Dir("./upload"))))

    //Listen on port 8080
    http.ListenAndServe(":"+port, nil)
}
