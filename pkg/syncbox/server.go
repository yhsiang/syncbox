package syncbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/apex/log"
)

var Host = "localhost"
var Port = "3000"
var WsUrl = fmt.Sprintf("ws://%s:%s", Host, Port)
var ServerUrl = fmt.Sprintf("http://%s:%s", Host, Port)

func NewServer(ctx context.Context, addr string, fileWatcher *FileWatcher) *SyncServer {
	server := NewSyncServer(ctx, addr, fileWatcher)
	queue := NewQueue()
	server.OnMessage(func(conn *SyncConnection, message []byte) {
		log.Infof("receive message %s", message)
		var msg Message
		err := json.Unmarshal(message, &msg)
		if err != nil {
			log.WithError(err).Error("failed to decode json")
		}

		switch msg.Command {
		case "syn":
			files := fileWatcher.Compare(msg.Files)
			conn.WriteJSON(Message{
				Command: "ack",
				Files:   files,
			})
			queue.push(files...)
		}
	})

	server.OnBinaryMessage(func(conn *SyncConnection, message []byte) {
		file := queue.pop()
		var fullPath = fmt.Sprintf("%s%s", fileWatcher.path, file.Path)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			log.WithError(err).Error("failed to read file")
			//TODO: handle http response
			return
		}

		var filepath = fmt.Sprintf("%s%s", fullPath, file.Name)
		f, err := os.OpenFile(filepath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			log.WithError(err).Error("failed to create file")
			//TODO: handle http response
			return
		}
		defer f.Close()

		io.Copy(f, bytes.NewReader(message))
	})

	fileWatcher.OnChange(func(files []File) {
		log.Infof("file changed %+v", files)
	})

	return server
}
