package syncbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yhsiang/syncbox/pkg/websocket"
)

//go:generate callbackgen -type SyncClient
type SyncClient struct {
	client      *websocket.WebSocketClient
	fileWatcher *FileWatcher

	fileChangeCallbacks []func(files []File)
}

func NewSyncClient(url string, fileWatcher *FileWatcher) *SyncClient {
	return &SyncClient{
		client:      websocket.New(url, http.Header{}),
		fileWatcher: fileWatcher,
	}
}

func (s *SyncClient) Connect(ctx context.Context) {
	s.client.SetReadTimeout(60 * time.Second)
	s.client.OnConnect(func(c *websocket.WebSocketClient) {
		fmt.Printf("connected to %s\n", s.client.Url)
	})

	s.client.OnMessage(func(m websocket.Message) {
		// TODO: handle response to upload / download files
	})

	s.OnFileChange(func(files []File) {
		// fmt.Println(files)
		var message = Message{
			Command: "sync",
			Files:   files,
		}
		out, err := json.Marshal(message)
		if err != nil {
			// log here
			return
		}

		if err := s.client.WriteJSON(out); err != nil {
			// log here
			fmt.Println(err)
		}
	})

	if err := s.client.Connect(ctx); err != nil {
		// logger.Errorf("failed to connect %+v", err)
	}
}

func (s *SyncClient) Disconnect() {
	s.client.Close()
}
