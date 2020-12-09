# Syncbox

Syncbox is a simple dropbox-like command line tool.

Client keeps detecting file changed and send it to server.

Server communicates with client by websocket, and receives files from http request.

# Usage

## Client
`$ go run ./cmd/syncbox /tmp/dropbox/client`

## Server
`$ go run ./cmd/syncboxd /tmp/dropbox/server`