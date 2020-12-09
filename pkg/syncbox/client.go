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

var uploadPath = strings.Join([]string{ServerUrl, "/upload"}, "")

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
		var msg Message
		err := json.Unmarshal(m.Body, &msg)
		if err != nil {
			log.WithError(err).Error("failed to decode json")
		}

		switch msg.Command {
		case "ack":
			for _, file := range msg.Files {
				err := s.uploadFile(file)
				if err != nil {
					log.WithError(err).Error("failed to upload")
					continue
				}
			}
		}
	})

	s.OnFileChange(func(files []File) {
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
	fileData, err := os.Open(strings.Join([]string{s.fileWatcher.path, file.Path}, ""))
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

	if fw, err = w.CreateFormField("path"); err != nil {
		return err
	}

	if _, err = io.Copy(fw, strings.NewReader(file.Path)); err != nil {
		return err
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
