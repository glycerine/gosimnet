package gosimnet

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/glycerine/idem"
)

const (
	// UserMaxPayload is the maximum network message
	// Write will send at once. For larger
	// sends, simply make multipe net.Conn.Write calls,
	// or use one of the io helpers.
	UserMaxPayload = 1_200_000
)

var lastSerialPrivate int64

// ErrShutdown is returned when the
// network or node goes down in the
// middle of an operation.
var ErrShutdown = fmt.Errorf("shutting down")

type localRemoteAddr interface {
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

// Client simulates a network
// client that can Dial out
// to a single Server.
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

// NewClient makes a new Client. Its name
// will double as its network address.
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

// NewClient makes a new Server. Its name
// will double as its network address.
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

// Server simulates a server process
// that can Accept connections from
// many Clients.
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

// Close shuts down the gosimnet network.
func (s *Net) Close() error {
	return s.simnetRendezvous.singleSimnet.Close()
}

// NewNet creates a new instance of a
// gosimnet network simulation.
// Clients and Servers from
// different Net can never see or
// hear from each other.
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

// AlterNode lets you simulate network and
// server node failures.
// The alter setting can be one of SHUTDOWN,
// PARTITION, UNPARTITION, RESTART.
func (s *Server) AlterNode(alter Alteration) {
	s.simnet.alterNode(s.simnode, alter)
}

// AlterNode lets you simulate network and
// client node failures.
// The alter setting can be one of SHUTDOWN,
// PARTITION, UNPARTITION, RESTART.
func (s *Client) AlterNode(alter Alteration) {
	s.simnet.alterNode(s.simnode, alter)
}

// Dial connects a Client to a Server.
func (c *Client) Dial(network, address string) (nc net.Conn, err error) {

	//vv("Client.Dial called with local='%v', server='%v'", c.name, address)

	err = c.runSimNetClient(c.name, address)

	select {
	case <-c.connected:
		nc = c.simconn
		return
	case <-c.halt.ReqStop.Chan:
		err = ErrShutdown
		return
	}
	return
}

// Close terminates the Client,
// moving it to SHUTDOWN state.
func (s *Client) Close() error {
	//vv("Client.Close running")

	if s.simnode == nil {
		return nil // not an error to Close before we started.
	}
	s.simnet.alterNode(s.simnode, SHUTDOWN)
	s.halt.ReqStop.Close()
	return nil
}

// LocalAddr retreives the local address that the
// Client is calling from.
func (c *Client) LocalAddr() string {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.cfg.localAddress
}

// RemoteAddr retreives the remote address for
// the Server that the Client is connected to.
func (c *Client) RemoteAddr() string {
	c.mut.Lock()
	defer c.mut.Unlock()

	return remote(c.conn)
}

// NewTimer makes a new Timer on the given Client.
// You must call ti.Discard() when done with it,
// or the simulation will leak that memory. It
// recommended to defer ti.Discard immediately.
func (c *Client) NewTimer(dur time.Duration) (ti *Timer) {
	ti = &Timer{
		isCli: true,
	}
	ti.simnet = c.simnet
	ti.simnode = c.simnode
	ti.simtimer = c.simnet.createNewTimer(c.simnode, dur, time.Now(), true) // isCli
	ti.C = ti.simtimer.timerC
	return
}

// Timer mocks the Go time.Timer object.
// Unlike Go timers, however, you must
// arrange to call Timer.Discard() when
// you are finished with the Timer.
// At the moment, Reset() is not implemented.
// Simply Discard the old Timer and create
// another using NewTimer.
type Timer struct {
	gotimer  *time.Timer
	isCli    bool
	simnode  *simnode
	simnet   *simnet
	simtimer *mop
	C        <-chan time.Time
}

// NewTimer makes a new Timer on the given Server.
// You must call ti.Discard() when done with it,
// or the simulation will leak that memory. It
// recommended to defer ti.Discard immediately.
func (s *Server) NewTimer(dur time.Duration) (ti *Timer) {
	ti = &Timer{
		isCli: false,
	}
	ti.simnet = s.simnet
	ti.simnode = s.simnode
	ti.simtimer = s.simnet.createNewTimer(s.simnode, dur, time.Now(), false) // isCli
	ti.C = ti.simtimer.timerC
	return
}

// Discard allows the gosimnet scheduler
// to dispose of an unneeded Timer. This
// is important to do manually in user code.
// Unlike the Go runtime, we do not have
// a garbage collector to clean up for us.
func (ti *Timer) Discard() (wasArmed bool) {
	if ti.simnet == nil {
		ti.gotimer.Stop()
		ti.gotimer = nil // Go will GC.
		return
	}
	wasArmed = ti.simnet.discardTimer(ti.simnode, ti.simtimer, time.Now())
	return
}
