package common

import (
	"encoding/csv"
	"os"
	"os/signal"
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

		agencyFile, err := os.Open(agencyFilePath)
		if err != nil {
			log.Criticalf("action: file_open | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}
		defer func(agencyFile *os.File) {
			closeErr := agencyFile.Close()
			if closeErr != nil {
				log.Criticalf("action: file_close | result: fail | client_id: %v | error: %v", c.config.ID, closeErr)
			}
		}(agencyFile)

		err = c.sendBets(c.config.ID, agencyFile)
		if err != nil {
			return
		}
	}
}

func (c *Client) sendBets(agencyId string, agencyFile *os.File) error {
	fileReader := csv.NewReader(agencyFile)
	currentBatch := make([]BetMessage, 0)
	var currentBatchSize int

	for {
		betLine, readErr := fileReader.Read()
		if readErr != nil {
			if len(currentBatch) > 0 {
				c.sendBatch(currentBatch)
			}
			break
		}

		currentBet := BetMessage{
			Agency:    agencyId,
			Firstname: betLine[0],
			Lastname:  betLine[1],
			Document:  betLine[2],
			Birthdate: betLine[3],
			Number:    betLine[4],
		}

		betSize := len(currentBet.Encode())

		// Me fijo que el batch no supere la cantidad maxima de apuestas
		// y que el tamaño del batch no supere el tamaño maximo permitido de 8kb
		// Me fijo si entra con o sin el separador, porque puede ser la apuesta final del batch
		if len(currentBatch) >= c.config.BatchMaxAmount ||
			(currentBatchSize+betSize+SeparatorBetsSize > MaxBatchSizeBytes && currentBatchSize+betSize > MaxBatchSizeBytes) {
			c.sendBatch(currentBatch)
			currentBatch = make([]BetMessage, 0)
			currentBatchSize = 0
		}

		currentBatch = append(currentBatch, currentBet)
		currentBatchSize += betSize + SeparatorBetsSize
	}

	return nil
}

func (c *Client) sendBatch(batch []BetMessage) {
	err := c.socket.Send(batch)
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
