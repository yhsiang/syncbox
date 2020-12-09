package syncbox

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/apex/log"
)

var Host = "localhost"
var Port = "3000"
var WsUrl = fmt.Sprintf("ws://%s:%s", Host, Port)
var ServerUrl = fmt.Sprintf("http://%s:%s", Host, Port)

func NewServer(ctx context.Context, addr string, fileWatcher *FileWatcher) *SyncServer {
	server := NewSyncServer(ctx, addr, fileWatcher)
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
		}

	})

	fileWatcher.OnChange(func(files []File) {
		log.Infof("file changed %+v", files)
	})

	return server
}
