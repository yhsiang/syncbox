package syncbox

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

var Host = "localhost"
var Port = "3000"
var WsUrl = strings.Join([]string{"ws://", Host, ":", Port}, "")
var ServerUrl = strings.Join([]string{"http://", Host, ":", Port}, "")

func NewServer(ctx context.Context, addr string, fileWatcher *FileWatcher) *SyncServer {
	server := NewSyncServer(ctx, addr, fileWatcher)
	server.OnMessage(func(conn *SyncConnection, message []byte) {
		var msg Message
		fmt.Printf("%s\n", message)
		err := json.Unmarshal(message, &msg)
		if err != nil {
			// log error
			fmt.Println(err)
		}

		switch msg.Command {
		case "syn":
			fmt.Printf("%s\n", message)
			files := fileWatcher.Compare(msg.Files)
			conn.WriteJSON(Message{
				Command: "ack",
				Files:   files,
			})
		}

	})

	fileWatcher.OnChange(func(files []File) {
		for _, file := range files {
			fmt.Printf("%+v\n", file)
		}
		fmt.Println("=========")
	})

	return server
}
