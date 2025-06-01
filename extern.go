package gosimnet

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/glycerine/idem"
)

var lastSerialPrivate int64

// ErrShutdown2 is returned when the
// network or node goes down in the
// middle of an operation.
var ErrShutdown2 = fmt.Errorf("shutting down")

// ErrShutdown returns ErrShutdown2. It
// is function to make it easy to diagnose
// where the error came from, if need be.
func ErrShutdown() error {
	return ErrShutdown2
}

type localRemoteAddr interface {
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

// Client simulates a network
// client that can Dial out
// to a single Server.
type Client struct {
	cfg       *Config
	mut       sync.Mutex
	net       *SimNet
	simNetCfg *SimNetConfig
	name      string
	halt      *idem.Halter
	simnode   *simnode
	simnet    *simnet

	simconn *simconn
	conn    *simconn

	connected chan error
}

// NewClient makes a new Client. Its name
// will double as its network address.
func (s *SimNet) NewClient(name string) (cli *Client) {
	var cfg SimNetConfig
	if s.simNetCfg != nil {
		cfg = *s.simNetCfg
	}
	cli = &Client{
		net:       s,
		simNetCfg: &cfg,
		name:      name,
		simnet:    s.simnetRendezvous.singleSimnet,
		halt:      idem.NewHalter(),
		connected: make(chan error, 1),
	}
	cli.simnet.halt.AddChild(cli.halt)
	return
}

// NewServer makes a new Server. Its name
// will double as its network address.
func (s *SimNet) NewServer(name string) (srv *Server) {

	var cfg SimNetConfig
	if s.simNetCfg != nil {
		cfg = *s.simNetCfg
	}
	srv = &Server{
		net:       s,
		simNetCfg: &cfg,
		name:      name,
		halt:      idem.NewHalter(),
		boundCh:   make(chan net.Addr, 1),
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
	cfg                *Config
	simNetCfg          *SimNetConfig
	net                *SimNet
	name               string
	halt               *idem.Halter
	simnode            *simnode
	simnet             *simnet
	boundCh            chan net.Addr
	simNetAddr         *SimNetAddr
	boundAddressString string
}

// SimNet holds a single gosimnet network.
// Clients and Servers who want to talk must
// be created from the same instance of SimNet,
// which they will use to rendezvous; in
// addition to their addresses (names).
type SimNet struct {
	simnet

	simNetCfg        *SimNetConfig
	mut              sync.Mutex
	simnetRendezvous *simnetRendezvous
	localAddress     string

	//ClientDialToHostPort string
}

// NewSimNet creates a new instance of a
// gosimnet network simulation.
// Clients and Servers from
// different SimNet can never see or
// hear from each other.
func NewSimNet(cfg *SimNetConfig) (n *SimNet) {
	n = &SimNet{
		simNetCfg:        cfg,
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

// AlterHost lets you simulate network and
// server node failures.
// The alter setting can be one of SHUTDOWN,
// PARTITION, UNPARTITION, RESTART.
func (s *Server) AlterHost(alter Alteration) {
	s.simnet.AlterHost(s.simnode.name, alter)
}

// AlterHost lets you simulate network and
// client node failures.
// The alter setting can be one of SHUTDOWN,
// PARTITION, UNPARTITION, RESTART.
func (s *Client) AlterHost(alter Alteration) {
	s.simnet.AlterHost(s.simnode.name, alter)
}

/*
// Dial connects a Client to a Server.
func (c *Client) Dial(network, address string) (nc net.Conn, err error) {

	//vv("Client.Dial called with local='%v', server='%v'", c.name, address)

	err = c.runSimNetClient(c.name, address)

	select {
	case <-c.connected:
		nc = c.simconn
		return
	case <-c.halt.ReqStop.Chan:
		err = ErrShutdown()
		return
	}
	return
}
*/

// Close terminates the Client,
// moving it to SHUTDOWN state.
func (s *Client) Close() error {
	//vv("Client.Close running")

	if s.simnode == nil {
		return nil // not an error to Close before we started.
	}
	s.simnet.AlterHost(s.simnode.name, SHUTDOWN)
	s.halt.ReqStop.Close()
	return nil
}

// LocalAddr retreives the local address that the
// Client is calling from.
func (c *Client) LocalAddr() string {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.net.localAddress
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
// is recommended to defer ti.Discard immediately.
func (c *Client) NewTimer(dur time.Duration) (ti *SimTimer) {
	ti = &SimTimer{
		isCli: true,
	}
	ti.simnet = c.simnet
	ti.simnode = c.simnode
	ti.simtimer = c.simnet.createNewTimer(c.simnode, dur, time.Now(), true) // isCli
	ti.C = ti.simtimer.timerC
	return
}

// SimTimer mocks the Go time.Timer object.
// Unlike Go timers, however, you must
// arrange to call Timer.Discard() when
// you are finished with the Timer.
// At the moment, Reset() is not implemented.
// Simply Discard the old Timer and create
// another using NewTimer.
type SimTimer struct {
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
// is recommended to defer ti.Discard immediately.
func (s *Server) NewTimer(dur time.Duration) (ti *SimTimer) {
	ti = &SimTimer{
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
func (ti *SimTimer) Discard() (wasArmed bool) {
	if ti.simnet == nil {
		return
	}
	wasArmed = ti.simnet.discardTimer(ti.simnode, ti.simtimer, time.Now())
	return
}

// SimNetConfig provides control parameters.
type SimNetConfig struct {

	// The barrier is the synctest.Wait call
	// the lets the caller resume only when
	// all other goro are durably blocked.
	BarrierOff bool
}

// NewSimNetConfig should be called
// to get an initial SimNetConfig to
// set parameters.
func NewSimNetConfig() *SimNetConfig {
	return &SimNetConfig{}
}

type Config struct {
	QuietTestMode    bool
	simnetRendezvous *simnetRendezvous
	serverBaseID     string
	SimNetConfig     *SimNetConfig
}

// allow simnet to properly classify LoneCli vs autocli
// associted with server peers.
const auto_cli_recognition_prefix = "auto-cli-from-"
