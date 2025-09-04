package common

import (
	"encoding/csv"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	sigs   chan os.Signal
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	client := &Client{
		config: config,
		sigs:   make(chan os.Signal, 1),
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
		return err
	}
	c.socket = socket
	return nil
}

func (c *Client) StartClientLoop(agencyFilePath string) {
	signal.Notify(c.sigs, syscall.SIGTERM)
	defer close(c.sigs)

	select {
	case <-c.sigs:
		c.sigtermHandler()
		return
	default:
		if c.createClientSocket() != nil {
			return
		}
		defer func(socket *ClientSocket) {
			err := socket.Close()
			if err != nil {
				log.Errorf("action: close_socket | result: fail | client_id: %v | error: %v", c.config.ID, err)
				return
			} else {
				log.Infof("action: close_socket | result: success | client_id: %v", c.config.ID)
				c.socket = nil
			}
		}(c.socket)

		agencyFile, err := os.Open(agencyFilePath)
		if err != nil {
			log.Errorf("action: file_open | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}
		defer func(agencyFile *os.File) {
			closeErr := agencyFile.Close()
			if closeErr != nil {
				log.Errorf("action: file_close | result: fail | client_id: %v | error: %v", c.config.ID, closeErr)
				return
			} else {
				log.Infof("action: file_close | result: success | client_id: %v", c.config.ID)
				c.socket = nil
			}
		}(agencyFile)

		c.sendBets(c.config.ID, agencyFile)
	}
}

func (c *Client) sendBets(agencyId string, agencyFile *os.File) {
	fileReader := csv.NewReader(agencyFile)
	currentBatch := make([]BetMessage, 0)
	var currentBatchSize int

	for {
		select {
		case <-c.sigs:
			c.sigtermHandler()
			return
		default:
			betLine, readErr := fileReader.Read()
			if readErr != nil {
				if len(currentBatch) > 0 {
					c.sendBatch(currentBatch, true)
				}
				log.Infof("action: send_bets | result: success | client_id: %v", c.config.ID)
				return
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
				c.sendBatch(currentBatch, false)
				currentBatch = make([]BetMessage, 0)
				currentBatchSize = 0
				time.Sleep(20 * time.Millisecond)
			}

			currentBatch = append(currentBatch, currentBet)
			currentBatchSize += betSize + SeparatorBetsSize
		}
	}
}

func (c *Client) sendBatch(batch []BetMessage, isFinalBatch bool) {
	err := c.socket.Send(batch, isFinalBatch)
	if err != nil {
		log.Errorf("action: batch_enviado | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	log.Infof("action: batch_enviado | result: success | client_id: %v", c.config.ID)
}

func (c *Client) sigtermHandler() {
	log.Infof("action: shutdown | result: in_progress | client_id: %v | msg: SIGTERM received", c.config.ID)
	if c.socket != nil {
		err := c.socket.Close()
		if err != nil {
			log.Errorf("action: shutdown_close_socket | result: fail | client_id: %v | error: %v", c.config.ID, err)
			log.Errorf("action: shutdown | result: fail | client_id: %v", c.config.ID)
			return
		} else {
			log.Infof("action: shutdown_close_socket | result: success | client_id: %v", c.config.ID)
		}
	}
	log.Infof("action: shutdown | result: success | client_id: %v", c.config.ID)
	return
}
