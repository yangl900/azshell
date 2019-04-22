package ws

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "", log.LUTC)
}

// Interface exposes the websocket interface
type Interface interface {
	Send(req []byte) error
	ReadChannel() chan []byte
	Closed() chan int
}

// Channel wraps a websocket connection.
type Channel struct {
	conn    *websocket.Conn
	receive chan []byte
	config  Config
}

// Config containers the configuratinos for the websocket channel
type Config struct {
	ConnectRetryWaitDuration time.Duration
	SendReceiveBufferSize    int
	URL                      string
}

func (c *Config) validateConfig() error {
	if c.URL == "" {
		return fmt.Errorf("websocket: must provide an address to connect to")
	}

	return nil
}

// NewWebsocketChannel creates a new channel to send and receive messages
// over the websocket.
func NewWebsocketChannel(config Config) (*Channel, error) {
	if err := config.validateConfig(); err != nil {
		return nil, err
	}

	c := &Channel{
		receive: make(chan []byte, config.SendReceiveBufferSize),
		config:  config,
	}

	c.connect()

	go c.setupReceiveChannel()

	return c, nil
}

// ReadChannel returns a channel to read from the websocket connection
func (c *Channel) ReadChannel() chan []byte {
	return c.receive
}

// Send sends a mesage over the websocket connection
func (c *Channel) Send(msg []byte) error {
	w, err := c.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return err
	}

	w.Write(msg)
	err = w.Close()
	return err
}

func (c *Channel) connect() {
	var conn *websocket.Conn
	var err error

	// try to connect to the web socket with retry
	for {
		conn, _, err = websocket.DefaultDialer.Dial(c.config.URL, nil)
		if err == nil {
			break
		}

		logger.Printf("failed to connect to websocket: %s with error :%v", c.config.URL, err)

		time.Sleep(c.config.ConnectRetryWaitDuration)
	}

	logger.Printf("Connected to %s", c.config.URL)

	if c.conn != nil {
		c.conn.Close()
	}

	c.conn = conn
}

func (c *Channel) setupReceiveChannel() {
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				logger.Printf("websocket connection closed: %v", err)
			} else {
				logger.Printf("failed to read message with error: %v. Close socket.", err)
			}
			close(c.receive)
			break
		}

		c.receive <- message
	}
}
