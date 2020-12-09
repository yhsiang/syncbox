package syncbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/yhsiang/syncbox/pkg/websocket"
)

var uploadPath = fmt.Sprintf("%s/upload", ServerUrl)
var downloadPath = fmt.Sprintf("%s/download", ServerUrl)

//go:generate callbackgen -type SyncClient
type SyncClient struct {
	client      *websocket.WebSocketClient
	fileWatcher *FileWatcher
	httpClient  *http.Client

	fileChangeCallbacks []func(files []File)
}

func NewSyncClient(url string, fileWatcher *FileWatcher) *SyncClient {
	return &SyncClient{
		client:      websocket.New(url, http.Header{}),
		fileWatcher: fileWatcher,
		httpClient:  &http.Client{},
	}
}

func (s *SyncClient) Connect(ctx context.Context) {
	s.client.SetReadTimeout(60 * time.Second)
	s.client.OnConnect(func(c *websocket.WebSocketClient) {
		fmt.Printf("connected to %s\n", s.client.Url)
	})

	s.client.OnMessage(func(m websocket.Message) {
		log.Infof("receive message %s", m.Body)
		var msg Message
		err := json.Unmarshal(m.Body, &msg)
		if err != nil {
			log.WithError(err).Error("failed to decode json")
		}

		switch msg.Command {
		case "ack":
			for _, file := range msg.Files {
				switch file.Action {
				case "upload":
					err := s.uploadFile(file)
					if err != nil {
						log.WithError(err).Error("failed to upload")
					}
				case "download":
					err := s.downloadFile(file)
					if err != nil {
						log.WithError(err).Error("failed to download")
					}
				}

			}
		}
	})

	s.OnFileChange(func(files []File) {
		log.Infof("file changed %+v", files)
		var message = Message{
			Command: "syn",
			Files:   files,
		}

		if err := s.client.WriteJSON(message); err != nil {
			log.WithError(err).Error("failed to send json")
		}
	})

	if err := s.client.Connect(ctx); err != nil {
		log.WithError(err).Error("failed to connect")
	}
}

func (s *SyncClient) Disconnect() {
	s.client.Close()
}

func (s *SyncClient) uploadFile(file File) error {
	var b bytes.Buffer
	var fw io.Writer
	w := multipart.NewWriter(&b)
	fileData, err := os.Open(fmt.Sprintf("%s%s", s.fileWatcher.path, file.FullName()))
	defer fileData.Close()
	if err != nil {
		return err
	}

	if fw, err = w.CreateFormFile("file", file.Name); err != nil {
		return err
	}

	if _, err = io.Copy(fw, fileData); err != nil {
		return err
	}

	var formData = make(map[string]string)
	formData["path"] = file.Path
	formData["filename"] = file.Name
	for key, value := range formData {
		if fw, err = w.CreateFormField(key); err != nil {
			log.WithError(err).Error("failed to create field")
			continue
		}

		if _, err = io.Copy(fw, strings.NewReader(value)); err != nil {
			log.WithError(err).Error("failed to add value")
			continue
		}
	}
	w.Close()

	req, err := http.NewRequest("POST", uploadPath, &b)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	res, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}

	return nil
}

func (s *SyncClient) downloadFile(file File) error {
	var url = fmt.Sprintf("%s/%s", downloadPath, file.ID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var fullPath = fmt.Sprintf("%s%s", s.fileWatcher.path, file.Path)
	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		return err
	}

	var filepath = fmt.Sprintf("%s%s", s.fileWatcher.path, file.FullName())
	f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	file.RootPath = s.fileWatcher.path
	err = file.CalChecksum()
	if err != nil {
		return err
	}

	s.fileWatcher.Set(file)
	return nil
}
