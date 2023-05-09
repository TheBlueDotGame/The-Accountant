package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/transaction"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

const (
	hubInnerChannelsBufferSize      = 100
	socketTickerInterval            = 100 * time.Millisecond
	socketContextExp                = 5 * time.Second
	socketWriteWait                 = 10 * time.Second
	socketPongWait                  = 20 * time.Second
	socketPingPeriod                = (socketPongWait * 4) / 5
	socketReadBufferSize            = 5012
	socketWriteBufferSize           = socketReadBufferSize * 256
	socketMaxMessageSize            = socketWriteBufferSize * 4
	clientMessageChannelsBufferSize = 512
	validatorsCountLimit            = 100
)

const (
	CommandEcho           = "echo"
	CommandSocketList     = "socketlist"
	CommandNewBlock       = "command_new_block"
	CommandNewTransaction = "command_new_transaction"
)

// Message is the message that is used to exchange information between
// the server and the client.
type Message struct {
	Command     string                  `json:"command"`               // Command is the command that refers to the action handler in websocket protocol.
	Error       string                  `json:"error,omitempty"`       // Error is the error message that is sent to the client.
	Block       block.Block             `json:"block,omitempty"`       // Block is the block that is sent to the client.
	Transaction transaction.Transaction `json:"transaction,omitempty"` // Transaction is the transaction validated by the central server and will be added to the next block.
	Sockets     []string                `json:"sockets,omitempty"`     // sockets is the list of central nodes web-sockets addresses.
}

type socket struct {
	address string
	hub     *hub
	conn    *websocket.Conn
	send    chan []byte
	repo    Repository
	log     logger.Logger
}

func (s *server) wsWrapper(ctx context.Context, c *fiber.Ctx) error {
	h := c.GetReqHeaders()

	token, ok := h["Token"]
	if !ok || token == "" {
		s.log.Error(
			fmt.Sprintf("websocket server, no token provided from address: %s", c.IP()))
		return fiber.ErrForbidden
	}

	if ok, err := s.repo.CheckToken(c.Context(), token); !ok || err != nil {
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to check token: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("token: %s provided in request by %s do not exists", token, c.IP()))
		return fiber.ErrForbidden
	}

	addr, ok := h["Address"]
	if !ok || addr == "" {
		s.log.Error(
			fmt.Sprintf("websocket server, no address provided from address: %s", c.IP()))
		return fiber.ErrForbidden
	}

	if ok, err := s.repo.CheckAddressExists(c.Context(), addr); err != nil || !ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("address %s does not exist in the repository", addr))
		return fiber.ErrForbidden
	}

	signatureString, ok := h["Signature"]
	if !ok || signatureString == "" {
		s.log.Error(
			fmt.Sprintf("websocket server, no signature provided from address: %s", c.IP()))
		return fiber.ErrForbidden
	}

	hashString, ok := h["Hash"]
	if !ok || hashString == "" {
		s.log.Error(
			fmt.Sprintf("websocket server, no signature provided from address: %s", c.IP()))
		return fiber.ErrForbidden
	}

	signature, err := hex.DecodeString(signatureString)
	if err != nil {
		s.log.Error(
			fmt.Sprintf("websocket server, signature not in hex format, provided from address: %s", c.IP()))
		return fiber.ErrForbidden
	}

	hash, err := hex.DecodeString(hashString)
	if err != nil {
		s.log.Error(
			fmt.Sprintf("websocket server, hash not in hex format, provided from address: %s", c.IP()))
		return fiber.ErrForbidden
	}

	var digest [32]byte
	copy(digest[:], hash)
	if err := s.bookkeeping.VerifySignature([]byte(token), signature, digest, addr); err != nil {
		s.log.Error(
			fmt.Sprintf("websocket server, signature validation failed from address: %s", c.IP()))
		return fiber.ErrForbidden
	}

	client := &socket{
		address: addr,
		hub:     s.hub,
		conn:    nil,
		send:    make(chan []byte, clientMessageChannelsBufferSize),
		repo:    s.repo,
		log:     s.log,
	}

	ctxx, cancel := context.WithCancel(ctx)
	serveWs := func(conn *websocket.Conn) {
		client.conn = conn
		client.hub.register <- client
		go client.writePump(ctxx, cancel)
		client.readPump(ctxx, cancel)
	}
	s.log.Info(fmt.Sprintf("websocket server, new connection from address: %s accepted", c.IP()))

	return websocket.New(serveWs)(c)
}

func (c *socket) readPump(ctx context.Context, cancel context.CancelFunc) {
	c.conn.SetReadLimit(socketMaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(socketPongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(socketPongWait)); return nil })

	tc := time.NewTicker(socketTickerInterval)
	defer tc.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tc.C:
			var msg Message
			err := c.conn.ReadJSON(&msg)
			if err != nil {
				switch {
				case websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure):
					c.log.Info(fmt.Sprintf("socket closing connection to the client %s due to unexpected error %s\n", c.address, err))
				default:
					c.log.Info(fmt.Sprintf("socket closing connection to the client %s due to error %s\n", c.address, err))
				}
				cancel()
				return
			}
			c.process(ctx, &msg)
		}
	}
}

func (c *socket) writePump(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(socketPingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.unregister <- c
		err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "central node stopped"))
		if err != nil {
			c.log.Error(fmt.Sprintf("central node write closing msg error, %s", err.Error()))
		}
		c.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case raw, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(socketWriteWait))
			if !ok {
				c.log.Info(fmt.Sprintf("socket closing connection to the client %s due to channel close", c.address))
				cancel()
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, raw); err != nil {
				c.log.Error(fmt.Sprintf("socket closing connection to the client %s due to %s", c.address, err))
				cancel()
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(socketWriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte(c.address)); err != nil {
				c.log.Error(fmt.Sprintf("socket closing connection to the client %s due to %s", c.address, err))
				cancel()
				return
			}
		}
	}
}

type hub struct {
	clients    map[string]*socket
	broadcast  chan *Message
	register   chan *socket
	unregister chan *socket
	log        logger.Logger
}

func newHub(log logger.Logger) *hub {
	return &hub{
		broadcast:  make(chan *Message, hubInnerChannelsBufferSize),
		register:   make(chan *socket, hubInnerChannelsBufferSize),
		unregister: make(chan *socket, hubInnerChannelsBufferSize),
		clients:    make(map[string]*socket, hubInnerChannelsBufferSize),
		log:        log,
	}
}

func (h *hub) run(ctx context.Context) {
outer:
	for {
		select {
		case client := <-h.register:
			if len(h.clients) >= validatorsCountLimit {
				client.conn.WriteMessage(websocket.CloseMessage, []byte("Max number of validators reached."))
				continue
			}
			h.clients[client.address] = client
		case client := <-h.unregister:
			delete(h.clients, client.address)
		case message := <-h.broadcast:
			raw, err := json.Marshal(&message)
			if err != nil {
				h.log.Error(fmt.Sprintf("hub failed to marshal message: %s", err.Error()))
				continue outer
			}
			for _, client := range h.clients {
				client.send <- raw
			}
		case <-ctx.Done():
			for _, client := range h.clients {
				delete(h.clients, client.address)
			}
			break outer
		}
	}
}

func (c *socket) process(ctx context.Context, msg *Message) {
	switch msg.Command {
	case CommandEcho:
		if err := c.echo(ctx, msg); err != nil {
			c.sendCommand(setCommandError(msg, err))
		}
		c.sendCommand(msg)
	case CommandSocketList:
		if err := c.socketList(ctx, msg); err != nil {
			c.sendCommand(setCommandError(msg, err))
		}
		c.sendCommand(msg)

	default:
		c.log.Info(fmt.Sprintf("socket received unknown command %s", msg.Command))
		c.sendCommand(setCommandError(msg, fmt.Errorf("unknown command %s", msg.Command)))
	}
}

func setCommandError(msg *Message, err error) *Message {
	msg.Error = err.Error()
	return msg
}

func (c socket) sendCommand(msg *Message) {
	raw, err := json.Marshal(&msg)
	if err != nil {
		c.log.Error(fmt.Sprintf("socket failed to marshal message: %s", err.Error()))
		return
	}
	c.send <- raw
}

func (c socket) broadcastCommend(msg *Message) {
	c.hub.broadcast <- msg
}

func (c *socket) echo(_ context.Context, msg *Message) error {
	return nil
}

func (c *socket) socketList(ctx context.Context, msg *Message) error {
	sockets, err := c.repo.ReadRegisteredNodesAddresses(ctx)
	if err != nil {
		return err
	}
	msg.Sockets = sockets
	return nil
}
