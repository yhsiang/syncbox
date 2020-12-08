package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	// "github.com/apex/log"
	"github.com/gorilla/websocket"
)

const DefaultWriteTimeout = 30 * time.Second
const DefaultReadTimeout = 30 * time.Second

var ErrReconnectContextDone = errors.New("reconnect canceled due to context done")
var ErrReconnectFailed = errors.New("failed to reconnect")
var ErrConnectionLost = errors.New("connection lost")

// var logger = log.WithFields(log.Fields{
// 	"component": "websocket",
// })

type Message struct {
	Type int
	Body []byte
}

//go:generate callbackgen -type WebSocketClient
type WebSocketClient struct {
	Url           string
	conn          *websocket.Conn
	Dialer        *websocket.Dialer
	requestHeader http.Header

	messageCallbacks    []func(m Message)
	connectCallbacks    []func(client *WebSocketClient)
	disconnectCallbacks []func(client *WebSocketClient)

	cancel       func()
	mu           sync.Mutex
	connected    bool
	readTimeout  time.Duration
	writeTimeout time.Duration
	pingInterval time.Duration

	backoff Backoff
	ping    func()
}

func New(url string, requestHeader http.Header) *WebSocketClient {
	return &WebSocketClient{
		Url:           url,
		Dialer:        websocket.DefaultDialer,
		readTimeout:   DefaultReadTimeout,
		writeTimeout:  DefaultWriteTimeout,
		requestHeader: requestHeader,
	}
}

func (c *WebSocketClient) setConn(conn *websocket.Conn) {
	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.mu.Unlock()
	c.EmitConnect(c)
}

func (c *WebSocketClient) SetPing(ping func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ping = ping
}

func (c *WebSocketClient) SetReadTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readTimeout = timeout
}

func (c *WebSocketClient) SetWriteTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeTimeout = timeout
}

func (c *WebSocketClient) SetPingInterval(interval time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pingInterval = interval
}

func (c *WebSocketClient) Reconnect(ctx context.Context) {
	if err := c.reconnect(ctx); err != nil {
		if err == ErrReconnectContextDone {
			return
		}

		c.Reconnect(ctx)
	}
}

func (c *WebSocketClient) Connect(basectx context.Context) error {
	// maintain a context by the client it self, so that we can manually shutdown the connection
	ctx, cancel := context.WithCancel(basectx)
	c.cancel = cancel

	conn, _, err := c.Dialer.DialContext(ctx, c.Url, c.requestHeader)
	if err == nil {
		// setup connection only when connected
		c.setConn(conn)
	}

	// 1) if connection is built up, start listening for messages.
	// 2) if connection is NOT ready, start reconnecting infinitely.
	go c.listen(ctx)

	return err
}

func (c *WebSocketClient) reconnect(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ErrReconnectContextDone
	default:
	}

	// log.Warnf("reconnecting x %d to %q", c.backoff.Attempt()+1, c.Url)
	conn, _, err := c.Dialer.DialContext(ctx, c.Url, c.requestHeader)
	if err != nil {
		dur := c.backoff.Duration()
		// logger.Warnf("failed to dial %s: %v, response: %+v. Wait for %v", c.Url, err, resp, dur)
		time.Sleep(dur)
		return ErrReconnectFailed
	}

	// logger.Infof("reconnected to %q", c.Url)
	c.backoff.Reset()
	c.setConn(conn)

	return nil
}

func (c *WebSocketClient) readMessages() error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return ErrConnectionLost
	}
	timeout := c.readTimeout
	conn := c.conn
	c.mu.Unlock()

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return err
	}

	msgtype, message, err := conn.ReadMessage()
	if err != nil {
		return err
	}

	c.EmitMessage(Message{msgtype, message})
	return nil
}

func (c *WebSocketClient) listen(ctx context.Context) {
	go c.keepalive(ctx)

	var pingNum = 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := c.readMessages(); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure) {
					// logger.Warnf("unexpected close error reconnecting: %v", err)
					c.SetDisconnected()
					c.Reconnect(ctx)
					continue
				}

				// logger.Warnf("failed to read message. error: %+v", err)
				if c.ping == nil || pingNum > 0 {
					c.SetDisconnected()
					pingNum = 0
					c.Reconnect(ctx)
				} else {
					c.ping()
					pingNum++
					time.Sleep(5 * time.Second)
				}
			}
		}
	}
}

func (c *WebSocketClient) keepalive(ctx context.Context) {
	c.mu.Lock()
	pingInterval := c.pingInterval
	c.mu.Unlock()
	if pingInterval == 0 {
		pingInterval = c.readTimeout / 2
	}

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// ensure connection is ready before pinging
			c.mu.Lock()
			if !c.connected {
				c.mu.Unlock()
				continue
			}
			conn := c.conn
			c.mu.Unlock()

			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(c.writeTimeout)); err != nil {
				// logger.WithError(err).Warnf("failed to write ping message")
			}
		}
	}
}

func (c *WebSocketClient) SetDisconnected() {
	c.mu.Lock()
	closed := false
	if c.conn != nil {
		closed = true
		c.conn.Close()
	}
	c.connected = false
	c.conn = nil
	c.mu.Unlock()

	if closed {
		c.EmitDisconnect(c)
	}
}

func (c *WebSocketClient) Close() (err error) {
	c.mu.Lock()
	// leave the listen goroutine before we close the connection
	// checking nil is to handle calling "Close" before "Connect" is called
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()
	c.SetDisconnected()

	return err
}

func (c *WebSocketClient) WriteJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return ErrConnectionLost
	}
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	err1 := json.NewEncoder(w).Encode(v)
	err2 := w.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (c *WebSocketClient) WriteTextMessage(message []byte) error {
	return c.WriteMessage(websocket.TextMessage, message)
}

func (c *WebSocketClient) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return ErrConnectionLost
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return err
	}
	return c.conn.WriteMessage(messageType, data)
}
