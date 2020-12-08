package syncbox

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
)

type syncHandler struct {
	context context.Context
	server  *SyncServer
}

func (h *syncHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawConn, err := h.upgradeConn(w, r)
	if err != nil {
		return
	}

	var ctx = r.Context()
	var conn = &SyncConnection{
		Conn:    rawConn,
		context: ctx,
		server:  h.server,
	}

	if err := conn.read(ctx); err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
			return
		}
	}
}

func (h *syncHandler) upgradeConn(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	return upgrader.Upgrade(w, r, nil)
}
