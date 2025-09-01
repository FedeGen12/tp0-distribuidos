package common

import (
	"bufio"
	"fmt"
	"net"
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

func (s *ClientSocket) Send(msg string) error {
	socketWriter := bufio.NewWriter(s.conn)
	msgBytes := []byte(msg)
	socketWriter.Write(msgBytes)

	if err := socketWriter.Flush(); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}
