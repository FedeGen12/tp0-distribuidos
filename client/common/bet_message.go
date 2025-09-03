package common

import (
	"encoding/csv"
	"os"
	"strings"
)

const SEPARATOR = ","

type BetMessage struct {
	Agency    string
	Firstname string
	Lastname  string
	Document  string
	Birthdate string
	Number    string
}

func (m BetMessage) Encode() []byte {
	fields := []string{
		m.Agency,
		m.Firstname,
		m.Lastname,
		m.Document,
		m.Birthdate,
		m.Number,
	}

	return []byte(strings.Join(fields, SEPARATOR))
}

func ParseBets(agencyId string, agencyFilePath string) []BetMessage {
	agencyFile, err := os.Open(agencyFilePath)
	if err != nil {
		log.Criticalf("action: file_open | result: fail | client_id: %v | error: %v", agencyId, err)
		return nil
	}
	defer func(agencyFile *os.File) {
		closeErr := agencyFile.Close()
		if closeErr != nil {
			log.Criticalf("action: file_close | result: fail | client_id: %v | error: %v", agencyId, closeErr)
		}
	}(agencyFile)

	return obtainBets(agencyId, agencyFile)
}

func obtainBets(agencyId string, agencyFile *os.File) []BetMessage {
	fileReader := csv.NewReader(agencyFile)
	bets := make([]BetMessage, 0)

	for {
		betLine, readErr := fileReader.Read()
		if readErr != nil {
			break
		}

		bets = append(bets, BetMessage{
			Agency:    agencyId,
			Firstname: betLine[0],
			Lastname:  betLine[1],
			Document:  betLine[2],
			Birthdate: betLine[3],
			Number:    betLine[4],
		})
	}

	return bets
}

func CreateBetBatches(bets []BetMessage, maxAmount int) [][]BetMessage {
	batches := make([][]BetMessage, 0)
	currentBatch := make([]BetMessage, 0)
	var currentBatchSize int

	for _, bet := range bets {
		encoded := bet.Encode()
		betSize := len(encoded)

		// Me fijo que el batch no supere la cantidad maxima de apuestas
		// y que el tamaño del batch no supere el tamaño maximo permitido de 8kb
		// Me fijo si entra con o sin el separador, porque puede ser la apuesta final del batch
		if len(currentBatch) >= maxAmount ||
			(currentBatchSize+betSize+SeparatorBetsSize > MaxBatchSizeBytes && currentBatchSize+betSize > MaxBatchSizeBytes) {
			batches = append(batches, currentBatch)
			currentBatch = make([]BetMessage, 0)
			currentBatchSize = 0
		}

		currentBatch = append(currentBatch, bet)
		currentBatchSize += betSize + SeparatorBetsSize
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}
