package gosimnet

import (
	"sync/atomic"
)

// Message is not needed by gosimnet users,
// but is exported to maintain compatability
// with the upstream code.
type Message struct {
	Serial  int64  `zid:"0"`
	JobSerz []byte `zid:"1"`

	// for emulating a socket connection,
	// after the JobSerz bytes are read,
	// is this the end-of-file?
	// Doubles as TCP RST (reset socket)
	// at the moment.
	EOF bool `zid:"2"`
}

func (m *Message) CopyForSimNetSend() (c *Message) {
	return &Message{
		Serial:  atomic.AddInt64(&lastSerialPrivate, 1),
		JobSerz: append([]byte{}, m.JobSerz...),
		EOF:     m.EOF,
	}
}

// NewMessage is not needed by gosimnet users,
// but is exported to maintain compatability
// with the upstream code.
func NewMessage() *Message {
	return &Message{}
}
