package gosimnet

import (
	//"fmt"
	"net"
	//"sync"
	"time"

	//"github.com/glycerine/idem"
	rpc "github.com/glycerine/rpc25519"
)

const (
	// UserMaxPayload is the maximum size in bytes
	// for network messages. This does not restrict
	// the user from doing io.Copy() to send a larger
	// stream, of course. It is mostly an internal net.Conn
	// implementation detail, but being aware of it
	// may allow user code to optimize their Writes.
	// For larger sends, simply make multipe net.Conn.Write calls,
	// advancing the slice by the returned written count
	// each time; or, as above, wrap your slice in a bytes.Buffer
	// and use io.Copy().
	UserMaxPayload = rpc.UserMaxPayload // 1_200_000 bytes, at this writing.
)

var lastSerialPrivate int64

// ErrShutdown2 is returned when the
// network or node goes down in the
// middle of an operation.
//var ErrShutdown2 = fmt.Errorf("shutting down")
//var ErrShutdown2 = rpc.ErrShutdown2

// ErrShutdown returns ErrShutdown2. It
// is function to make it easy to diagnose
// where the error came from, if need be.
//func ErrShutdown() error {
//	return ErrShutdown2
//}

/*type localRemoteAddr interface {
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}
*/

// SimClient simulates a network
// client that can Dial out
// to a single SimServer.
type SimClient struct {
	name   string
	cfg    *rpc.Config
	rpcCli *rpc.Client

	// mut       sync.Mutex
	// net       *SimNet
	// simNetCfg *SimNetConfig
	// name      string
	// halt      *idem.Halter
	// simnode   *simnode
	// simnet    *simnet

	// simconn *simnetConn
	// conn    *simnetConn

	// connected chan error
}

// NewClient makes a new Client. Its name
// will double as its network address.
func (s *SimNet) NewSimClient(name string) (cli *SimClient, err error) {

	//var rpcCli *rpc.Client
	cloneCfg := *s.cfg

	// rpcCli, err = rpc.NewClient(name, &cloneCfg)
	// if err != nil {
	// 	return
	// }

	cli = &SimClient{
		name: name,
		cfg:  &cloneCfg,
		//rpcCli: rpcCli,
	}
	return
	// var cfg SimNetConfig
	// if s.simNetCfg != nil {
	// 	cfg = *s.simNetCfg
	// }
	// cli = &SimClient{
	// 	net:       s,
	// 	simNetCfg: &cfg,
	// 	name:      name,
	// 	simnet:    s.simnetRendezvous.singleSimnet,
	// 	halt:      idem.NewHalter(),
	// 	connected: make(chan error, 1),
	// }
	// cli.simnet.halt.AddChild(cli.halt)
	//return
}

// NewSimServer makes a new SimServer. Its name
// will double as its network address.
func (s *SimNet) NewSimServer(name string) (srv *SimServer) {

	cloneCfg := *s.cfg

	rpcSrv := rpc.NewServer(name, &cloneCfg)
	srv = &SimServer{
		name:   name,
		cfg:    &cloneCfg,
		rpcSrv: rpcSrv,
	}

	// var cfg SimNetConfig
	// if s.simNetCfg != nil {
	// 	cfg = *s.simNetCfg
	// }
	// srv = &SimServer{
	// 	net:       s,
	// 	simNetCfg: &cfg,
	// 	name:      name,
	// 	halt:      idem.NewHalter(),
	// 	boundCh:   make(chan net.Addr, 1),
	// }

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
type SimServer struct {
	cfg    *rpc.Config
	name   string
	rpcSrv *rpc.Server

	// mut                sync.Mutex
	// simNetCfg          *SimNetConfig
	// net                *SimNet
	// name               string
	// halt               *idem.Halter
	// simnode            *simnode
	// simnet             *simnet
	// boundCh            chan net.Addr
	// simNetAddr         *SimNetAddr
	// boundAddressString string
}

// SimNet holds a single gosimnet network.
// Clients and Servers who want to talk must
// be created from the same instance of SimNet,
// which they will use to rendezvous; in
// addition to their addresses (names).
type SimNet struct {
	cfg *rpc.Config
	net *rpc.Simnet

	// simNetCfg        *SimNetConfig
	// mut              sync.Mutex
	// simnetRendezvous *simnetRendezvous
	// localAddress     string

	//ClientDialToHostPort string
}

// Close shuts down the gosimnet network.
func (s *SimNet) Close() error {
	if s.net != nil {
		s.net.Close()
	}
	return nil
}

// NewSimNet creates a new instance of a
// gosimnet network simulation.
// Clients and Servers from
// different SimNet can never see or
// hear from each other.
func NewSimNet(config *rpc.Config) (n *SimNet) {

	var cfg *rpc.Config
	if config != nil {
		clone := *config
		cfg = &clone
	} else {
		cfg = rpc.NewConfig()
	}
	cfg.UseSimNet = true

	n = &SimNet{
		cfg: cfg,
	}
	return
}

/*
// AlterNode lets you simulate network and
// server node failures.
// The alter setting can be one of SHUTDOWN,
// PARTITION, UNPARTITION, RESTART.
func (s *SimServer) AlterNode(alter Alteration) {
	s.simnet.alterNode(s.simnode, alter)
}

// AlterNode lets you simulate network and
// client node failures.
// The alter setting can be one of SHUTDOWN,
// PARTITION, UNPARTITION, RESTART.
func (s *SimClient) AlterNode(alter Alteration) {
	s.simnet.alterNode(s.simnode, alter)
}
*/

// Dial connects a Client to a Server.
func (c *SimClient) Dial(network, address string) (nc net.Conn, err error) {

	//vv("Client.Dial called with local='%v', server='%v'", c.name, address)

	c.cfg.ClientDialToHostPort = address
	c.rpcCli, err = rpc.NewClient(c.name, c.cfg)
	if err != nil {
		return
	}

	err = c.rpcCli.Start()
	if err != nil {
		return
	}

	return c.rpcCli.GetSimconn()
}

// Close terminates the Client,
// moving it to SHUTDOWN state.
func (s *SimClient) Close() error {
	//vv("Client.Close running")
	return s.rpcCli.Close()
}

// LocalAddr retreives the local address that the
// Client is calling from.
func (c *SimClient) LocalAddr() string {
	if c.rpcCli == nil {
		return ""
	}
	return c.rpcCli.LocalAddr()
}

// RemoteAddr retreives the remote address for
// the Server that the Client is connected to.
func (c *SimClient) RemoteAddr() string {
	if c.rpcCli == nil {
		return ""
	}
	return c.rpcCli.RemoteAddr()
}

// NewTimer makes a new Timer on the given Client.
// You must call ti.Discard() when done with it,
// or the simulation will leak that memory. It
// is recommended to defer ti.Discard immediately.
func (c *SimClient) NewTimer(dur time.Duration) (ti *rpc.SimTimer) {
	return c.rpcCli.NewTimer(dur)

	// ti = &SimTimer{
	// 	isCli: true,
	// }
	// ti.simnet = c.simnet
	// ti.simnode = c.simnode
	// ti.simtimer = c.simnet.createNewTimer(c.simnode, dur, time.Now(), true) // isCli
	// ti.C = ti.simtimer.timerC
	// return
}

// SimTimer mocks the Go time.Timer object.
// Unlike Go timers, however, you must
// arrange to call Timer.Discard() when
// you are finished with the Timer.
// At the moment, Reset() is not implemented.
// Simply Discard the old Timer and create
// another using NewTimer.
/*type SimTimer struct {
	gotimer  *time.Timer
	isCli    bool
	simnode  *simnode
	simnet   *simnet
	simtimer *mop
	C        <-chan time.Time
}*/

// NewTimer makes a new Timer on the given Server.
// You must call ti.Discard() when done with it,
// or the simulation will leak that memory. It
// is recommended to defer ti.Discard immediately.
func (s *SimServer) NewTimer(dur time.Duration) (ti *rpc.SimTimer) {
	return s.rpcSrv.NewTimer(dur)

	// ti = &SimTimer{
	// 	isCli: false,
	// }
	// ti.simnet = s.simnet
	// ti.simnode = s.simnode
	// ti.simtimer = s.simnet.createNewTimer(s.simnode, dur, time.Now(), false) // isCli
	// ti.C = ti.simtimer.timerC
	// return
}

/*
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
*/

// // SimNetConfig allows for future custom
// // settings of the gosimnet. The
// // NewSimNetConfig function should
// // be used to get an initial instance.
// type SimNetConfig struct {
// 	BarrierOff bool
// }

// NewSimNetConfig should be called
// to get an initial SimNetConfig to
// set parameters and pass to NewSimNet().
func NewSimNetConfig() *rpc.Config {
	return rpc.NewConfig()
}

// Listen currently ignores the network and addr strings,
// which are there to match the net.Listen method.
// The addr will be the name set on NewServer(name).
func (s *SimServer) Listen(network, addr string) (lsn net.Listener, err error) {

	return s.rpcSrv.Listen(network, addr)

	// // start the server, first server boots the network,
	// // but it can continue even if the server is shutdown.
	// addrCh := make(chan net.Addr, 1)
	// s.runSimNetServer(s.name, addrCh, s.simNetCfg)
	// lsn = s
	// var netAddr *SimNetAddr
	// select {
	// case netAddrI := <-addrCh:
	// 	netAddr = netAddrI.(*SimNetAddr)
	// case <-s.halt.ReqStop.Chan:
	// 	err = ErrShutdown()
	// }
	// _ = netAddr
	// return
}

// Close terminates the Server. Any blocked Accept
// operations will be unblocked and return errors.
func (s *SimServer) Close() error {
	return s.rpcSrv.Close()

	//vv("Server.Close() running")
	// s.mut.Lock()
	// defer s.mut.Unlock()
	// if s.simnode == nil {
	// 	return nil // not an error to Close before we started.
	// }
	// s.simnet.alterNode(s.simnode, SHUTDOWN)
	// //vv("simnet.alterNode(s.simnode, SHUTDOWN) done for %v", s.name)
	// s.halt.ReqStop.Close()
	// // nobody else we need ack from, so don't hang on:
	// //<-s.halt.Done.Chan
	// return nil
}
