package syncbox

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type downloadHandler struct {
	context     context.Context
	fileWatcher *FileWatcher
}

func (d *downloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file := "test.pdf"

	downloadBytes, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Println(err)
	}

	mime := http.DetectContentType(downloadBytes)

	fileSize := len(string(downloadBytes))

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", "attachment; filename="+file)
	w.Header().Set("Content-Length", strconv.Itoa(fileSize))

	http.ServeContent(w, r, file, time.Now(), bytes.NewReader(downloadBytes))

	return
}
