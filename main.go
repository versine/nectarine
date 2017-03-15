package main

import (
	"crypto/md5"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	http.Handle("/upload", handler(upload))
	http.Handle("/listen", handler(listen))
	http.Handle("/", handler(index))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type handler func(http.ResponseWriter, *http.Request) (status int, err error)

func (f handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	status, err := f(w, r)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(status), status)
		return
	}

	// if all is well, send the response
	fmt.Fprint(w)
}

func index(w http.ResponseWriter, r *http.Request) (status int, err error) {
	switch r.Method {
	case http.MethodGet:
		t, err := template.ParseFiles("template/index.html")
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to parse index template: %v", err)
		}
		if err := t.Execute(w, nil); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to execute index template: %v", err)
		}

		return http.StatusOK, nil
	default:
		return http.StatusBadRequest, fmt.Errorf("bad request to index")
	}
}

type fileEntry struct {
	Name string
	Size int64
}

func listen(w http.ResponseWriter, r *http.Request) (status int, err error) {
	switch r.Method {
	case http.MethodGet:
		t, err := template.ParseFiles("template/listen.html")
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to parse listen template: %v", err)
		}

		fileInfos, err := ioutil.ReadDir("./test")
		files := make([]fileEntry, len(fileInfos))
		for i, file := range fileInfos {
			files[i] = fileEntry{Name: file.Name(), Size: file.Size()}
		}

		if err := t.Execute(w, files); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to execute listen template: %v", err)
		}

		return http.StatusOK, nil
	default:
		return http.StatusBadRequest, fmt.Errorf("bad request to listen")
	}
}

func upload(w http.ResponseWriter, r *http.Request) (status int, err error) {
	switch r.Method {
	case http.MethodGet:
		crutime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(crutime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		t, err := template.ParseFiles("template/upload.html")
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to parse upload form template: %v", err)
		}
		if err := t.Execute(w, token); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to execute upload form template: %v", err)
		}

		return http.StatusOK, nil

	case http.MethodPost:
		maxSize := int64(32 << 20) // max file size
		r.ParseMultipartForm(maxSize)
		uploadedFile, header, err := r.FormFile("uploadfile")
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to retrieve file from POST request: %v", err)
		}
		defer uploadedFile.Close()

		localFile, err := os.OpenFile("./test/"+header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to open local file: %v", err)
		}
		defer localFile.Close()

		// write the upload to disk
		io.Copy(localFile, uploadedFile)

		return http.StatusCreated, nil
	default:
		return http.StatusBadRequest, fmt.Errorf("Bad request")
	}
}
