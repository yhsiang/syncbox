package syncbox

import (
	"context"
	"net/http"
)

//go:generate callbackgen -type SyncServer
type SyncServer struct {
	*http.Server

	messageCallbacks       []func(conn *SyncConnection, message []byte)
	binaryMessageCallbacks []func(conn *SyncConnection, message []byte)
	// uploadCallbacks []
}

func NewSyncServer(ctx context.Context, addr string, fileWatcher *FileWatcher) *SyncServer {
	var server = &SyncServer{
		Server: &http.Server{
			Addr: addr,
		},
	}

	var mux = http.NewServeMux()
	mux.Handle("/", &syncHandler{
		context: ctx,
		server:  server,
	})

	mux.Handle("/upload", &uploadHandler{
		context:     ctx,
		fileWatcher: fileWatcher,
	})

	server.Server.Handler = mux
	return server
}
