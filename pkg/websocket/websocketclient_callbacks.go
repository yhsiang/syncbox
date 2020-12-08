// Code generated by "callbackgen -type WebSocketClient"; DO NOT EDIT.

package websocket

import ()

func (c *WebSocketClient) OnMessage(cb func(m Message)) {
	c.messageCallbacks = append(c.messageCallbacks, cb)
}

func (c *WebSocketClient) EmitMessage(m Message) {
	for _, cb := range c.messageCallbacks {
		cb(m)
	}
}

func (c *WebSocketClient) OnConnect(cb func(client *WebSocketClient)) {
	c.connectCallbacks = append(c.connectCallbacks, cb)
}

func (c *WebSocketClient) EmitConnect(client *WebSocketClient) {
	for _, cb := range c.connectCallbacks {
		cb(client)
	}
}

func (c *WebSocketClient) OnDisconnect(cb func(client *WebSocketClient)) {
	c.disconnectCallbacks = append(c.disconnectCallbacks, cb)
}

func (c *WebSocketClient) EmitDisconnect(client *WebSocketClient) {
	for _, cb := range c.disconnectCallbacks {
		cb(client)
	}
}