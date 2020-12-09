package syncbox

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"

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
		return
	}
	defer file.Close()

	var path = strings.Join([]string{
		u.fileWatcher.path,
		r.FormValue("path"),
	}, "")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.WithError(err).Error("failed to create file")
		return
	}
	defer f.Close()
	io.Copy(f, file)

	//TODO: handle http response
	return
}
