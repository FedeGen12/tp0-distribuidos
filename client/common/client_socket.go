package common

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

const AmountBatchesSizeBytes = 4
const BatchSizeBytes = 4
const MaxBatchSizeBytes = 8192
const SeparatorBetsSize = 1 // Es por el caracter de separacion entre apuestas en el batch
const SeparatorBets = ";"

type ClientSocket struct {
	conn net.Conn
}

func BindCLientSocket(address string) (*ClientSocket, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &ClientSocket{conn}, nil
}

func (s *ClientSocket) Close() error {
	err := s.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

func (s *ClientSocket) Send(bets []BetMessage, batchMaxSize int) error {
	socketWriter := bufio.NewWriter(s.conn)

	betBatches := createBetBatches(bets, batchMaxSize)

	amountBatchesBytes := make([]byte, AmountBatchesSizeBytes)
	binary.BigEndian.PutUint32(amountBatchesBytes, uint32(len(betBatches)))
	_, writeErr := socketWriter.Write(amountBatchesBytes)
	if writeErr != nil {
		return fmt.Errorf("write error: %v", writeErr)
	}

	for _, batch := range betBatches {
		batchErr := s.sendBatch(batch, socketWriter)
		if batchErr != nil {
			return batchErr
		}
	}

	if err := socketWriter.Flush(); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}

func (s *ClientSocket) sendBatch(batch []BetMessage, socketWriter *bufio.Writer) error {
	encodedBets := make([][]byte, 0)

	for _, bet := range batch {
		encodedBets = append(encodedBets, bet.Encode())
	}

	batchBytes := bytes.Join(encodedBets, []byte(SeparatorBets))
	batchSizeBytes := make([]byte, BatchSizeBytes)
	binary.BigEndian.PutUint32(batchSizeBytes, uint32(len(batchBytes)))

	_, err1 := socketWriter.Write(batchSizeBytes)
	_, err2 := socketWriter.Write(batchBytes)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("write error: %v %v", err1, err2)
	}

	return nil
}

func createBetBatches(bets []BetMessage, maxAmount int) [][]BetMessage {
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
