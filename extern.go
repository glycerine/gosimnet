package gosimnet

import (
	"net"
	"time"

	rpc "github.com/glycerine/rpc25519"
)

// SimClient simulates a network
// client that can Dial out
// to a single SimServer.
type SimClient struct {
	name   string
	cfg    *rpc.Config
	rpcCli *rpc.Client
}

// NewClient makes a new Client. Its name
// doubles as its network address.
func (s *SimNet) NewSimClient(name string) (cli *SimClient, err error) {

	cloneCfg := *s.cfg

	// do we have to wait until Dial so we have the
	// server's address to contact? we might think it would
	// be fine, but changing the config after submit is
	// kind of tricky, so hold off until the Dial.
	//
	// var rpcCli *rpc.Client
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
}

// Dial connects a Client to a Server.
func (c *SimClient) Dial(network, address string) (nc net.Conn, err error) {

	//vv("Client.Dial called with local='%v', server='%v'", c.name, address)

	c.cfg.ClientDialToHostPort = address
	c.rpcCli, err = rpc.NewClient(c.name, c.cfg)
	if err != nil {
		return
	}
	return c.rpcCli.Dial(network, address)
}

// NewSimServer makes a new SimServer. Its name
// doubles as its network address.
func (s *SimNet) NewSimServer(name string) (srv *SimServer) {

	cloneCfg := *s.cfg

	rpcSrv := rpc.NewServer(name, &cloneCfg)
	srv = &SimServer{
		name:   name,
		cfg:    &cloneCfg,
		rpcSrv: rpcSrv,
	}
	return
}

// Server simulates a server process
// that can Accept connections from
// many Clients.
type SimServer struct {
	cfg    *rpc.Config
	name   string
	rpcSrv *rpc.Server
}

// SimNet holds a single gosimnet network.
// Clients and Servers who want to talk must
// be created from the same instance of SimNet,
// which they will use to rendezvous; in
// addition to their addresses (names).
//
// Note: the actual rpc.Simnet creation is
// lazy, since it is tied to the rpc.Server.Start
// process.
// Hence the SimNet will only really be available
// for network/fault modeling after the first
// NewServer() has returned from its first Server Listen
// (i.e. Start) call.
// Then GetSimnetSnapshot etc should succeed.
type SimNet struct {
	cfg *rpc.Config
	net *rpc.Simnet
}

// GetRpcSimnet gives access to the network
// modeling and fault-inject and snapshot API's
// from the underlying rpc25519 Simnet.
// Due to the lazy simnet instantiation,
// we must return nil until the first Server is
// started/listening on a socket, since
// before that there is no Simnet to return.
func (s *SimNet) GetRpcSimnet() *rpc.Simnet {
	if s.net == nil {
		s.net = s.cfg.GetSimnet()
	}
	return s.net // might still be nil.
}

func (s *SimNet) GetSimnetSnapshot() (snap *rpc.SimnetSnapshot) {
	if s.net == nil {
		s.net = s.cfg.GetSimnet()
	}
	if s.net != nil {
		return s.net.GetSimnetSnapshot()
	}
	return nil
}

// Close shuts down the gosimnet network.
func (s *SimNet) Close() error {
	if s.net == nil {
		s.net = s.cfg.GetSimnet()
	}
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
//
// Note: the actual rpc.Simnet creation is
// lazy, since it is tied to the rpc.Server.Start
// process.
// Hence the SimNet will only really be available
// for network/fault modeling after the first
// NewServer() has returned from its first Server Listen
// (i.e. Start) call.
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
}

// NewTimer makes a new Timer on the given Server.
// You must call ti.Discard() when done with it,
// or the simulation will leak that memory. It
// is recommended to defer ti.Discard immediately.
func (s *SimServer) NewTimer(dur time.Duration) (ti *rpc.SimTimer) {
	return s.rpcSrv.NewTimer(dur)
}

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
}

// Close terminates the Server. Any blocked Accept
// operations will be unblocked and return errors.
func (s *SimServer) Close() error {
	return s.rpcSrv.Close()
}
