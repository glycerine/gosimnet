package gosimnet

import (
// "fmt"
// "net"
// "time"
)

func (s *Client) setLocalAddr(conn localRemoteAddr) {
	s.mut.Lock()
	defer s.mut.Unlock()

	s.net.localAddress = local(conn)
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

	c.net.simnetRendezvous.singleSimnetMut.Lock()
	c.simnet = c.net.simnetRendezvous.singleSimnet
	c.net.simnetRendezvous.singleSimnetMut.Unlock()

	if c.simnet == nil {
		panic("arg. client could not find cfg.simnetRendezvous.singleSimnet")
	}

	//vv("runSimNetClient c.simnet = %p, '%v', goro = %v", c.simnet, c.name, GoroNumber())

	// ignore serverAddr in favor of cfg.ClientDialToHostPort
	// which tests actually set.

	if serverAddr == "" { // && c.net.ClientDialToHostPort == ""
		panic("gotta have a server address of some kind")
	}
	// c.net.ClientDialToHostPort vestigial?
	registration := c.simnet.newClientRegistration(c, localHostPort, serverAddr, serverAddr, c.cfg.serverBaseID)

	select {
	case c.simnet.cliRegisterCh <- registration:
	case <-c.simnet.halt.ReqStop.Chan:
		return ErrShutdown()
	case <-c.halt.ReqStop.Chan:
		return ErrShutdown()
	}

	select {
	case <-registration.done:
	case <-c.simnet.halt.ReqStop.Chan:
		return ErrShutdown()
	case <-c.halt.ReqStop.Chan:
		return ErrShutdown()
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
		return ErrShutdown()
	}
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
