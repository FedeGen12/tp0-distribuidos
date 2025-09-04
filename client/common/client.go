package common

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	sigs   chan os.Signal
}

// NewClient Initializes a new client receiving the configuration
// as a parameter
func NewClient(config ClientConfig) *Client {
	return &Client{config: config, sigs: make(chan os.Signal, 1)}
}

// CreateClientSocket Initializes client socket. In case of
// failure, error is printed in stdout/stderr and exit 1
// is returned
func (c *Client) createClientSocket() error {
	socket, err := BindCLientSocket(c.config.ServerAddress)
	if err != nil {
		log.Criticalf("action: connect | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return err
	}
	c.socket = socket
	return nil
}

func (c *Client) StartClientLoop(agencyFilePath string) {
	signal.Notify(c.sigs, syscall.SIGTERM, syscall.SIGINT)
	defer close(c.sigs)

	go func() {
		<-c.sigs
		c.signalHandler()
	}()

	if c.createClientSocket() != nil {
		return
	}
	defer func(socket *ClientSocket) {
		if c.socket != nil {
			err := socket.Close()
			c.socket = nil
			if err != nil {
				log.Errorf("action: close_socket | result: fail | client_id: %v | error: %v", c.config.ID, err)
				return
			} else {
				log.Infof("action: close_socket | result: success | client_id: %v", c.config.ID)
			}
		}
	}(c.socket)

	errId := c.sendClientId(c.config.ID)
	if errId != nil {
		return
	}

	agencyFile, err := os.Open(agencyFilePath)
	if err != nil {
		log.Criticalf("action: file_open | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	defer func(agencyFile *os.File) {
		if closeErr := agencyFile.Close(); closeErr != nil {
			log.Criticalf("action: file_close | result: fail | client_id: %v | error: %v", c.config.ID, closeErr)
			return
		}
	}(agencyFile)

	err = c.sendBets(c.config.ID, agencyFile)
	if err != nil {
		return
	}

	if err = c.socket.SendNotifyMessage(); err != nil {
		log.Errorf("action: notify_message | result: fail | client_id: %v | error: %v", c.config.ID, err)
		return
	}
	log.Infof("action: notify_message | result: success | client_id: %v", c.config.ID)

	winners, err := c.socket.RecvWinners()
	if err != nil {
		if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "use of closed network connection") {
			log.Infof("action: consulta_ganadores | result: shutdown")
			return
		}
		log.Errorf("action: consulta_ganadores | result: fail | error: %v", err)
		return
	}
	log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", len(winners))
}

func (c *Client) sendBets(agencyId string, agencyFile *os.File) error {
	fileReader := csv.NewReader(agencyFile)
	currentBatch := make([]BetMessage, 0)
	var currentBatchSize int

	for {
		betLine, readErr := fileReader.Read()
		if readErr != nil {
			if len(currentBatch) > 0 {
				return c.socket.Send(currentBatch)
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
		if c.shouldSendBatch(len(currentBatch), currentBatchSize, betSize) {
			if err := c.socket.Send(currentBatch); err != nil {
				return err
			}
			currentBatch = make([]BetMessage, 0)
			currentBatchSize = 0
		}

		currentBatch = append(currentBatch, currentBet)
		currentBatchSize += betSize + SeparatorBetsSize
	}

	return nil
}

func (c *Client) signalHandler() {
	log.Infof("action: shutdown | result: in_progress | client_id: %v | msg: SIGTERM received", c.config.ID)
	if c.socket != nil {
		err := c.socket.Close()
		c.socket = nil
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

func (c *Client) sendClientId(clientId string) error {
	parsedClientId, parseErr := strconv.Atoi(clientId)
	if parseErr != nil {
		return fmt.Errorf("action: parse_id | result: fail | client_id: %v | error: %v", clientId, parseErr)
	}

	err := c.socket.SendClientId(parsedClientId)
	if err != nil {
		return fmt.Errorf("action: send_id | result: fail | client_id: %v | error: %v", clientId, err)
	}
	log.Infof("action: send_id | result: success | client_id: %v", clientId)
	return nil
}

func (c *Client) shouldSendBatch(batchLen, batchSize, betSize int) bool {
	return batchLen >= c.config.BatchMaxAmount ||
		(batchSize+betSize+SeparatorBetsSize > MaxBatchSizeBytes && batchSize+betSize > MaxBatchSizeBytes)
}
