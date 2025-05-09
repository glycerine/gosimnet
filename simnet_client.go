package gosimnet

import (
	//"fmt"
	"net"
	"time"
)

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

func (s *Client) Close() error {
	//vv("Client.Close running")

	if s.simnode == nil {
		return nil // not an error to Close before we started.
	}
	s.simnet.alterNode(s.simnode, SHUTDOWN)
	s.halt.ReqStop.Close()
	return nil
}

func (s *Client) setLocalAddr(conn localRemoteAddr) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.cfg.localAddress = local(conn)
}

// LocalAddr retreives the local host/port that the
// Client is calling from.
func (c *Client) LocalAddr() string {
	c.mut.Lock()
	defer c.mut.Unlock()
	return c.cfg.localAddress
}

// RemoteAddr retreives the remote host/port for
// the Server that the Client is connected to.
func (c *Client) RemoteAddr() string {
	c.mut.Lock()
	defer c.mut.Unlock()

	return remote(c.conn)
}

func remote(nc localRemoteAddr) string {
	ra := nc.RemoteAddr()
	return ra.Network() + "://" + ra.String()
}

func local(nc localRemoteAddr) string {
	la := nc.LocalAddr()
	return la.Network() + "://" + la.String()
}

func (c *Client) runSimNetClient(localHostPort, serverAddr string) (err error) {

	//defer func() {
	//vv("runSimNetClient defer on exit running client = %p", c)
	//}()

	//netAddr := &SimNetAddr{network: "cli simnet@" + localHostPort}

	// how does client pass this to us?/if we need it at all?
	//simNetConfig := &SimNetConfig{}

	c.cfg.simnetRendezvous.singleSimnetMut.Lock()
	c.simnet = c.cfg.simnetRendezvous.singleSimnet
	c.cfg.simnetRendezvous.singleSimnetMut.Unlock()

	if c.simnet == nil {
		panic("arg. client could not find cfg.simnetRendezvous.singleSimnet")
	}

	//vv("runSimNetClient c.simnet = %p, '%v', goro = %v", c.simnet, c.name, GoroNumber())

	// ignore serverAddr in favor of cfg.ClientDialToHostPort
	// which tests actually set.

	if c.cfg.ClientDialToHostPort == "" && serverAddr == "" {
		panic("gotta have a server address of some kind")
	}
	// c.cfg.ClientDialToHostPort vestigial?
	registration := c.simnet.newClientRegistration(c, localHostPort, serverAddr, serverAddr)

	select {
	case c.simnet.cliRegisterCh <- registration:
	case <-c.simnet.halt.ReqStop.Chan:
		return ErrShutdown
	case <-c.halt.ReqStop.Chan:
		return ErrShutdown
	}

	select {
	case <-registration.done:
	case <-c.simnet.halt.ReqStop.Chan:
		return ErrShutdown
	case <-c.halt.ReqStop.Chan:
		return ErrShutdown
	}

	conn := registration.conn
	c.simnode = registration.simnode // == conn.local
	c.simconn = conn
	c.conn = conn

	// maybe if needed and no deadlock:
	c.setLocalAddr(conn)
	// tell user level client code we are ready
	//vv("client set local addr: '%v'", conn.LocalAddr())
	select {
	case c.connected <- nil:
	case <-c.halt.ReqStop.Chan:
		return ErrShutdown
	}
	return
}

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

type Timer struct {
	gotimer  *time.Timer
	isCli    bool
	simnode  *simnode
	simnet   *simnet
	simtimer *mop
	C        <-chan time.Time
}

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

func (ti *Timer) Discard() (wasArmed bool) {
	if ti.simnet == nil {
		ti.gotimer.Stop()
		ti.gotimer = nil // Go will GC.
		return
	}
	wasArmed = ti.simnet.discardTimer(ti.simnode, ti.simtimer, time.Now())
	return
}

/*
func (ti *Timer) Reset(dur time.Duration) (wasArmed bool) {
	if ti.simnet == nil {
		return ti.gotimer.Reset(dur)
	}
	wasArmed = ti.simnet.resetTimer(ti, time.Now(), ti.onCli)
	return
}
func (ti *Timer) Stop(dur time.Duration) (wasArmed bool) {
	if ti.simnet == nil {
		return ti.gotimer.Stop()
	}
	wasArmed = ti.simnet.stopTimer(ti, time.Now(), ti.onCli)
	return
}

// returns wasArmed (not expired or stopped)
func (c *Client) StopTimer(ti *Timer) bool {
	return ti.Stop()
}
func (s *Server) StopTimer(ti *Timer) bool {
	return ti.Stop()
}

// returns wasArmed (not expired or stopped)
func (c *Client) ResetTimer(ti *Timer, dur time.Duration) bool {
	return ti.Reset(dur)
}
func (s *Server) ResetTimer(ti *Timer, dur time.Duration) bool {
	return ti.Reset(dur)
}
*/
