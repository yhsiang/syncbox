package syncbox

import (
	"context"
	"net/http"
)

//go:generate callbackgen -type SyncServer
type SyncServer struct {
	*http.Server

	messageCallbacks []func(conn *SyncConnection, message []byte)
	// uploadCallbacks []
}

func NewSyncServer(ctx context.Context, addr string) *SyncServer {
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
	server.Server.Handler = mux
	return server
}
