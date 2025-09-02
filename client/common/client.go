package common

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID             string
	ServerAddress  string
	BatchMaxAmount int
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

func (c *Client) StartClientLoop(agencyFilePath string) {
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

		c.sendClientId(c.config.ID)

		bets := ParseBets(c.config.ID, agencyFilePath)
		if bets == nil {
			return
		}

		c.sendBets(bets)
		c.sendNotifyMessage()
		c.recvWinners()
	}
}

func (c *Client) sendBets(bets []BetMessage) {
	err := c.socket.Send(bets, c.config.BatchMaxAmount)
	if err != nil {
		log.Errorf("action: batch_enviado | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	log.Infof("action: batch_enviado | result: success | client_id: %v", c.config.ID)
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

func (c *Client) sendNotifyMessage() {
	err := c.socket.SendNotifyMessage()
	if err != nil {
		log.Errorf("action: notify_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	log.Infof("action: notify_message | result: success | client_id: %v", c.config.ID)
}

func (c *Client) sendClientId(clientId string) {
	parsedClientId, parseErr := strconv.Atoi(clientId)
	if parseErr != nil {
		log.Errorf("action: parse_id | result: fail | client_id: %v | error: %v", clientId, parseErr)
	}

	err := c.socket.SendClientId(parsedClientId)
	if err != nil {
		log.Errorf("action: send_id | result: fail | client_id: %v | error: %v", clientId, err)
	}
	log.Infof("action: send_id | result: success | client_id: %v", clientId)
}

func (c *Client) recvWinners() {
	winners, err := c.socket.RecvWinners()
	if err != nil {
		log.Errorf("action: consulta_ganadores | result: fail | error: %v", err)
		return
	}
	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", len(winners))
}
