package syncbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/apex/log"
)

type uploadHandler struct {
	context     context.Context
	fileWatcher *FileWatcher
}

func (u *uploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.NotFound(w, r)
		return
	}

	err := r.ParseMultipartForm(5 * 1024 * 1024)
	if err != nil {
		log.WithError(err).Error("failed to parse")
		//TODO: handle http response
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		log.WithError(err).Error("failed to read file")
		//TODO: handle http response
		return
	}
	defer file.Close()
	var fullPath = fmt.Sprintf("%s%s", u.fileWatcher.path, r.FormValue("path"))
	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		log.WithError(err).Error("failed to read file")
		//TODO: handle http response
		return
	}

	var filepath = fmt.Sprintf("%s%s", fullPath, r.FormValue("filename"))
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.WithError(err).Error("failed to create file")
		//TODO: handle http response
		return
	}
	defer f.Close()
	io.Copy(f, file)

	//TODO: handle http response
	return
}
