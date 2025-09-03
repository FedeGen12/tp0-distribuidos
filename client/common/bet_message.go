package common

import (
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
