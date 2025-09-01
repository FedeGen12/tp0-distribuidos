package common

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
)

const BetSize = 4

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

func (s *ClientSocket) Send(betMsg BetMessage) error {
	socketWriter := bufio.NewWriter(s.conn)
	msgBytes := betMsg.Encode()
	msgSizeBytes := make([]byte, BetSize)
	binary.BigEndian.PutUint32(msgSizeBytes, uint32(len(msgBytes)))

	_, err1 := socketWriter.Write(msgSizeBytes)
	_, err2 := socketWriter.Write(msgBytes)
	if err1 != nil || err2 != nil {
		return fmt.Errorf("write error: %v %v", err1, err2)
	}

	if err := socketWriter.Flush(); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}
