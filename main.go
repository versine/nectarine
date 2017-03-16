package main

import (
	"context"
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

type ctxKey int

const (
	passKey ctxKey = iota
)

func withAuth(password string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), passKey, password)

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

type handler func(http.ResponseWriter, *http.Request) (status int, err error)

func main() {
	password, set := os.LookupEnv("UPLOAD_SECRET")
	if !set {
		password = "p4ssw0rd"
	}

	http.Handle("/upload", withAuth(password, handler(upload)))
	http.Handle("/listen", handler(listen))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", handler(index))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

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
	Size string // filesize in MB
}

func listen(w http.ResponseWriter, r *http.Request) (status int, err error) {
	switch r.Method {
	case http.MethodGet:
		t, err := template.ParseFiles("template/listen.html")
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to parse listen template: %v", err)
		}

		fileInfos, err := ioutil.ReadDir("./static")
		files := make([]fileEntry, len(fileInfos))
		for i, file := range fileInfos {
			sizeString := fmt.Sprintf("%.2f", float64(file.Size())/(1024.0*1024.0))
			files[i] = fileEntry{Name: file.Name(), Size: sizeString}
		}

		if err := t.Execute(w, files); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to execute listen template: %v", err)
		}

		return http.StatusOK, nil
	default:
		return http.StatusBadRequest, fmt.Errorf("bad request to listen")
	}
}

func renderUploadTemplate(w http.ResponseWriter, r *http.Request, goodStatus int) (status int, err error) {
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

	return goodStatus, nil
}

func upload(w http.ResponseWriter, r *http.Request) (status int, err error) {
	switch r.Method {
	case http.MethodGet:
		return renderUploadTemplate(w, r, http.StatusOK)

	case http.MethodPost:
		maxSize := int64(32 << 20) // max file size
		r.ParseMultipartForm(maxSize)

		pass, ok := r.Context().Value(passKey).(string)
		if !ok || pass != r.PostFormValue("password") {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w)
			return
		}

		uploadedFile, header, err := r.FormFile("uploadfile")
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to retrieve file from POST request: %v", err)
		}
		defer uploadedFile.Close()

		localFile, err := os.OpenFile("./static/"+header.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to open local file: %v", err)
		}
		defer localFile.Close()

		// write the upload to disk
		io.Copy(localFile, uploadedFile)

		return renderUploadTemplate(w, r, http.StatusCreated)
	default:
		return http.StatusBadRequest, fmt.Errorf("bad request")
	}
}
