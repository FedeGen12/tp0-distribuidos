package common

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

const AmountBatchesSizeBytes = 4
const BatchSizeBytes = 4
const MaxBatchSizeBytes = 8192
const SeparatorBetsSize = 1 // Es por el caracter de separacion entre apuestas en el batch
const SeparatorBets = ";"
const AmountWinnersSizeBytes = 4
const DocumentSizeBytes = 4

const (
	BatchMessage  = 0
	NotifyMessage = 1
	IdMessage     = 2
)

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

	betBatches := CreateBetBatches(bets, batchMaxSize)

	_, typeMsgErr := socketWriter.Write([]byte{BatchMessage})
	if typeMsgErr != nil {
		return typeMsgErr
	}

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

func (s *ClientSocket) SendNotifyMessage() error {
	writer := bufio.NewWriter(s.conn)

	_, writeErr := writer.Write([]byte{NotifyMessage})
	if writeErr != nil {
		return writeErr
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to send notify message : %v", err)
	}

	return nil
}

func (s *ClientSocket) SendClientId(clientId int) error {
	writer := bufio.NewWriter(s.conn)

	_, writeErr := writer.Write([]byte{IdMessage})
	if writeErr != nil {
		return writeErr
	}

	_, writeErr2 := writer.Write([]byte{byte(clientId)})
	if writeErr2 != nil {
		return writeErr2
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to send client ID: %v", err)
	}

	return nil
}

func (s *ClientSocket) RecvWinners() ([]int, error) {
	amountWinnersBytes := make([]byte, AmountWinnersSizeBytes)

	_, readErr := io.ReadFull(s.conn, amountWinnersBytes)
	if readErr != nil {
		return nil, fmt.Errorf("failed to receive winners amount: %v", readErr)
	}

	amountWinners := binary.BigEndian.Uint32(amountWinnersBytes)

	winnersBytes := make([]byte, amountWinners*DocumentSizeBytes)

	_, readErr2 := io.ReadFull(s.conn, winnersBytes)
	if readErr2 != nil {
		return nil, fmt.Errorf("failed to receive winners: %v", readErr2)
	}

	winnersDocuments := make([]int, 0, amountWinners)
	for i := 0; i < len(winnersBytes); i += DocumentSizeBytes {
		document := binary.BigEndian.Uint32(winnersBytes[i : i+DocumentSizeBytes])
		winnersDocuments = append(winnersDocuments, int(document))
	}

	return winnersDocuments, nil
}
