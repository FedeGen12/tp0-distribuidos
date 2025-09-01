package common

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
}

// Client Entity that encapsulates how
type Client struct {
	config ClientConfig
	socket *ClientSocket
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
	}
	return client
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	socket, err := BindCLientSocket(c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
	}
	c.socket = socket
	return nil
}

func (c *Client) StartClientLoop() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM)
	defer close(sigs)

	select {
	case <-sigs:
		c.sigtermHandler()
		return
	default:
		if c.createClientSocket() != nil {
			return
		}
		defer func(socket *ClientSocket) {
			err := socket.Close()
			if err != nil {
				log.Criticalf(
					"action: close socket | result: fail | client_id: %v | error: %v",
					c.config.ID,
					err,
				)
			}
		}(c.socket)

		c.sendBet()
	}
}

func (c *Client) sendBet() {
	msg := BetMessage{
		Agency:    c.config.ID,
		Firstname: os.Getenv("NOMBRE"),
		Lastname:  os.Getenv("APELLIDO"),
		Document:  os.Getenv("DOCUMENTO"),
		Birthdate: os.Getenv("NACIMIENTO"),
		Number:    os.Getenv("NUMERO"),
	}

	log.Infof("data: %v", msg)

	err := c.socket.Send(msg)
	if err != nil {
		log.Errorf("action: apuesta_enviada | result: fail | error: %v", err)
		return
	}
	log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", msg.Document, msg.Number)
}

func (c *Client) sigtermHandler() {
	log.Infof("action: shutdown | result: success | client_id: %v | msg: SIGTERM received", c.config.ID)
	if c.socket != nil {
		err := c.socket.Close()
		if err != nil {
			log.Criticalf(
				"action: close socket | result: fail | client_id: %v | error: %v",
				c.config.ID,
				err,
			)
			return
		}
	}
	return
}
