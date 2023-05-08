package validator

import (
	"context"

	"github.com/fasthttp/websocket"
)

type socket struct {
	conn   *websocket.Conn
	cancel context.CancelFunc
}
