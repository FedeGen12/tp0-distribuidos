package common

import (
	"encoding/csv"
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
	return &Client{config: config}
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
			if err := socket.Close(); err != nil {
				log.Criticalf("action: close socket | result: fail | client_id: %v | error: %v", c.config.ID, err)
			}
		}(c.socket)

		c.sendClientId(c.config.ID)

		agencyFile, err := os.Open(agencyFilePath)
		if err != nil {
			log.Criticalf("action: file_open | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}
		defer func(agencyFile *os.File) {
			if closeErr := agencyFile.Close(); closeErr != nil {
				log.Criticalf("action: file_close | result: fail | client_id: %v | error: %v", c.config.ID, closeErr)
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
			log.Errorf("action: consulta_ganadores | result: fail | error: %v", err)
			return
		}
		log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %v", len(winners))
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

func (c *Client) sigtermHandler() {
	log.Infof("action: shutdown | result: success | client_id: %v | msg: SIGTERM received", c.config.ID)
	if c.socket != nil {
		if err := c.socket.Close(); err != nil {
			log.Criticalf("action: close socket | result: fail | client_id: %v | error: %v", c.config.ID, err)
			return
		}
	}
	return
}

func (c *Client) sendClientId(clientId string) {
	parsedClientId, parseErr := strconv.Atoi(clientId)
	if parseErr != nil {
		log.Errorf("action: parse_id | result: fail | client_id: %v | error: %v", clientId, parseErr)
		return
	}

	if err := c.socket.SendClientId(parsedClientId); err != nil {
		log.Errorf("action: send_id | result: fail | client_id: %v | error: %v", clientId, err)
		return
	}
	log.Infof("action: send_id | result: success | client_id: %v", clientId)
}

func (c *Client) shouldSendBatch(batchLen, batchSize, betSize int) bool {
	return batchLen >= c.config.BatchMaxAmount ||
		(batchSize+betSize+SeparatorBetsSize > MaxBatchSizeBytes && batchSize+betSize > MaxBatchSizeBytes)
}
