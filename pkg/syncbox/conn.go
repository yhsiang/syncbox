package syncbox

import (
	"context"
	"sync"

	"github.com/gorilla/websocket"
)

type SyncConnection struct {
	mu sync.Mutex
	*websocket.Conn
	context context.Context
	server  *SyncServer
}

func (c *SyncConnection) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

			// case msg, ok := <-c.pubsub.Channel():
			// 	if !ok {
			// 		continue
			// 	}

			// 	c.mu.Lock()
			// 	err := c.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			// 	c.mu.Unlock()
			// 	if err != nil {
			// 		c.server.EmitError(err)
			// 	}
		}
	}
}

// read handles messages from client and send it to messageCallbacks of server.
func (c *SyncConnection) read(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, message, err := c.ReadMessage()
		if err != nil {
			return err
		}

		if len(message) == 0 {
			continue
		}

		// c.server.EmitMessage(c, message)
	}
}
