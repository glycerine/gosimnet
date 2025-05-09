package gosimnet

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/glycerine/idem"
)

const (
	// UserMaxPayload is the maximum network message
	// Write will send at once.
	UserMaxPayload = 1_200_000
)

var lastSerialPrivate int64
var ErrShutdown = fmt.Errorf("shutting down")

type localRemoteAddr interface {
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

type Client struct {
	mut     sync.Mutex
	cfg     *Net
	name    string
	halt    *idem.Halter
	simnode *simnode
	simnet  *simnet

	simconn *simnetConn
	conn    *simnetConn

	connected chan error
}

func (s *Net) NewClient(name string) (cli *Client) {
	cli = &Client{
		cfg:       s,
		name:      name,
		simnet:    s.simnetRendezvous.singleSimnet,
		halt:      idem.NewHalter(),
		connected: make(chan error, 1),
	}
	cli.simnet.halt.AddChild(cli.halt)
	return
}

func (s *Net) NewServer(name string) (srv *Server) {
	srv = &Server{
		cfg:     s,
		name:    name,
		halt:    idem.NewHalter(),
		boundCh: make(chan net.Addr, 1),
	}
	// We can't add link up the halters yet, do it
	// in simnet_server.go, runSimNetServer();
	// we lazily boot up the network when
	// the first server in it is brought up
	// so the gosimnet doesn't really exist
	// until then.
	//
	//srv.simnet.halt.AddChild(srv.halt)

	return
}

type Server struct {
	mut                sync.Mutex
	cfg                *Net
	name               string
	halt               *idem.Halter
	simnode            *simnode
	simnet             *simnet
	boundCh            chan net.Addr
	netAddr            *SimNetAddr
	boundAddressString string
}

// Net holds a single gosimnet network.
// Clients and Servers who want to talk must
// be provided the same instance of Net,
// which they will use to rendezvous; in
// addition to their addresses.
type Net struct {
	mut              sync.Mutex
	simnetRendezvous *simnetRendezvous
	localAddress     string

	ClientDialToHostPort string
}

func (s *Net) Close() error {
	return s.simnetRendezvous.singleSimnet.Close()
}

func NewNet() (n *Net) {
	n = &Net{
		simnetRendezvous: &simnetRendezvous{},
	}
	return
}

// gotta have just one simnet, shared by all the
// clients and servers in a single gosimnet.
type simnetRendezvous struct {
	singleSimnetMut sync.Mutex
	singleSimnet    *simnet
}

type Message struct {
	Serial  int64  `zid:"0"`
	JobSerz []byte `zid:"1"`
}

func (m *Message) CopyForSimNetSend() (c *Message) {
	return &Message{
		Serial:  atomic.AddInt64(&lastSerialPrivate, 1),
		JobSerz: append([]byte{}, m.JobSerz...),
	}
}

func NewMessage() *Message {
	return &Message{}
}
