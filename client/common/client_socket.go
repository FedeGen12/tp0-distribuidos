package common

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

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

func (s *ClientSocket) Send(batch []BetMessage) error {
	socketWriter := bufio.NewWriter(s.conn)

	batchErr := s.sendBatch(batch, socketWriter)
	if batchErr != nil {
		return batchErr
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
