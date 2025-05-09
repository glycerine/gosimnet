package gosimnet

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	//"sync/atomic"
	"time"

	"github.com/glycerine/idem"
)

// Server implements net.Listener

// Accept waits for and returns the next connection to the listener.
func (s *Server) Accept() (nc net.Conn, err error) {
	select {
	case nc = <-s.simnode.tellServerNewConnCh:
		if IsNil(nc) {
			err = ErrShutdown
			return
		}
		//vv("Server.Accept returning nc = '%#v'", nc.(*simnetConn))
	case <-s.halt.ReqStop.Chan:
		err = ErrShutdown
	}
	return
}

func (s *Server) Addr() (a net.Addr) {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.netAddr
}

func (s *Server) Listen() (netAddr *SimNetAddr, err error) {
	// start the server, first server boots the network,
	// but it can continue even if the server is shutdown.
	addrCh := make(chan *SimNetAddr, 1)
	s.runSimNetServer(s.name, addrCh, nil)

	select {
	case netAddr = <-addrCh:
	case <-s.halt.ReqStop.Chan:
		err = ErrShutdown
	}
	return
}

// Any blocked Accept operations will be unblocked and return errors.
func (s *Server) Close() error {
	//vv("Server.Close() running")
	s.mut.Lock()
	defer s.mut.Unlock()
	if s.simnode == nil {
		return nil // not an error to Close before we started.
	}
	s.simnet.alterNode(s.simnode, SHUTDOWN)
	//vv("simnet.alterNode(s.simnode, SHUTDOWN) done for %v", s.name)
	s.halt.ReqStop.Close()
	// nobody else we need ack from, so don't hang on:
	//<-s.halt.Done.Chan
	return nil
}

func (s *Server) runSimNetServer(serverAddr string, boundCh chan *SimNetAddr, simNetConfig *SimNetConfig) {

	// satisfy uConn interface; don't crash cli/tests that check
	netAddr := &SimNetAddr{network: "gosimnet", addr: serverAddr, name: s.name, isCli: false}

	// idempotent, so all new servers can try;
	// only the first will boot it up (still pass s for s.halt);
	// second and subsequent will get back the
	// cfg.simnetRendezvous.singleSimnet, which is a
	// per config shared simnet.
	simnet := s.cfg.bootSimNetOnServer(simNetConfig, s)

	//deadlock? yes. arg. simnet.halt.AddChild(s.halt)
	s.cfg.mut.Lock()
	s.cfg.simnetRendezvous.singleSimnet = simnet
	s.cfg.mut.Unlock()

	// sets s.simnode, s.simnet as side-effect
	serverNewConnCh, err := simnet.registerServer(s, netAddr)
	if err != nil {
		if err == ErrShutdown {
			//vv("simnet_server sees shutdown in progress")
			return
		}
		panicOn(err)
	}
	if serverNewConnCh == nil {
		panic(fmt.Sprintf("%v got a nil serverNewConnCh, should not be allowed!", s.name))
	}
	// we don't need to otherwise save serverNewConnCh, since
	// s.simnode.tellServerNewConnCh already has it.

	s.mut.Lock() // avoid data races
	addrs := netAddr.Network() + "://" + netAddr.String()
	s.boundAddressString = addrs
	s.netAddr = netAddr
	s.mut.Unlock()

	if boundCh != nil {
		select {
		case boundCh <- netAddr:
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// a connection between two nodes.
// implements uConn, see simnet_server.go
// the readMessage/sendMessage are well tested;
// the other net.Conn generic Read/Write less so, at the moment.
type simnetConn struct {
	mut sync.Mutex

	owner *simnode

	// distinguish cli from srv
	isCli   bool
	net     *simnet
	netAddr *SimNetAddr // local address

	local  *simnode
	remote *simnode

	readDeadlineTimer *mop
	sendDeadlineTimer *mop

	nextRead []byte

	// no more reads, but serve the rest of nextRead.
	localClosed  *idem.IdemCloseChan
	remoteClosed *idem.IdemCloseChan
}

func newSimnetConn() *simnetConn {
	return &simnetConn{
		localClosed:  idem.NewIdemCloseChan(),
		remoteClosed: idem.NewIdemCloseChan(),
	}
}

// originally not actually used much by simnet. We'll
// fill them out to try and allow testing of net.Conn code.
// doc:
// "Write writes len(p) bytes from p to the
// underlying data stream. It returns the number
// of bytes written from p (0 <= n <= len(p)) and
// any error encountered that caused the write to
// stop early. Write must return a non-nil error
// if it returns n < len(p). Write must not modify
// the slice data, even temporarily.
// Implementations must not retain p."
//
// Implementation note: we will only send
// UserMaxPayload bytes at a time.
func (s *simnetConn) Write(p []byte) (n int, err error) {
	s.mut.Lock()
	defer s.mut.Unlock()

	isCli := s.isCli

	if s.localClosed.IsClosed() {
		err = &simconnError{
			desc: "use of closed network connection",
		}
		return
	}

	if len(p) == 0 {
		return
	}

	msg := NewMessage()
	n = len(p)
	if n > UserMaxPayload {
		n = UserMaxPayload
		// copy into the "kernel buffer"
		msg.JobSerz = append([]byte{}, p[:n]...)
	} else {
		msg.JobSerz = append([]byte{}, p...)
	}

	var sendDead chan time.Time
	if s.sendDeadlineTimer != nil {
		sendDead = s.sendDeadlineTimer.timerC
	}

	send := newSendMop(msg, isCli)
	send.origin = s.local
	send.fileLine = fileLine(3)
	send.target = s.remote
	send.initTm = time.Now()

	//vv("top simnet.Write(%v) from %v at %v to %v", string(msg.JobSerz), send.origin.name, send.fileLine, send.target.name)

	select {
	case s.net.msgSendCh <- send:
	case <-s.net.halt.ReqStop.Chan:
		n = 0
		err = ErrShutdown
		return
	case timeout := <-sendDead:
		_ = timeout
		n = 0
		err = &simconnError{isTimeout: true, desc: "i/o timeout"}
		return
	case <-s.localClosed.Chan:
		n = 0
		err = io.EOF
		return
	case <-s.remoteClosed.Chan:
		n = 0
		err = io.EOF
		return
	}

	//vv("net has it, about to wait for proceed... simnetConn.Write('%v') isCli=%v, origin=%v ; target=%v;", string(send.msg.JobSerz), s.isCli, send.origin.name, send.target.name)

	select {
	case <-send.proceed:
		return
	case <-s.net.halt.ReqStop.Chan:
		n = 0
		err = ErrShutdown
		return
	case timeout := <-sendDead:
		_ = timeout
		n = 0
		err = &simconnError{isTimeout: true, desc: "i/o timeout"}
		return
	case <-s.localClosed.Chan:
		n = 0
		err = io.EOF
		return
	case <-s.remoteClosed.Chan:
		n = 0
		err = io.EOF
		return
	}
	return
}

// doc:
// "When Read encounters an error or end-of-file
// condition after successfully reading n > 0 bytes,
// it returns the number of bytes read. It may
// return the (non-nil) error from the same call
// or return the error (and n == 0) from a subsequent call.
// An instance of this general case is that a
// Reader returning a non-zero number of bytes
// at the end of the input stream may return
// either err == EOF or err == nil. The next
// Read should return 0, EOF.
// ...
// "If len(p) == 0, Read should always
// return n == 0. It may return a non-nil
// error if some error condition is known,
// such as EOF.
//
// "Implementations of Read are discouraged
// from returning a zero byte count with a nil
// error, except when len(p) == 0. Callers should
// treat a return of 0 and nil as indicating that
// nothing happened; in particular it
// does not indicate EOF.
//
// "Implementations must not retain p."
func (s *simnetConn) Read(data []byte) (n int, err error) {

	if len(data) == 0 {
		return
	}

	s.mut.Lock()
	defer s.mut.Unlock()

	// leftovers?
	if len(s.nextRead) > 0 {
		n = copy(data, s.nextRead)
		s.nextRead = s.nextRead[n:]
		return
	}

	if s.localClosed.IsClosed() {
		err = io.EOF
		return
	}

	isCli := s.isCli

	var readDead chan time.Time
	if s.readDeadlineTimer != nil {
		readDead = s.readDeadlineTimer.timerC
	}

	read := newReadMop(isCli)
	read.initTm = time.Now()
	read.origin = s.local
	read.fileLine = fileLine(2)
	read.target = s.remote

	//vv("in simnetConn.Read() isCli=%v, origin=%v at %v; target=%v", s.isCli, read.origin.name, read.fileLine, read.target.name)

	select {
	case s.net.msgReadCh <- read:
	case <-s.net.halt.ReqStop.Chan:
		err = ErrShutdown
		return
	case timeout := <-readDead:
		_ = timeout
		err = os.ErrDeadlineExceeded
		//err = &simconnError{isTimeout: true, desc: "i/o timeout"}
		return
	case <-s.localClosed.Chan:
		err = io.EOF
		return
	case <-s.remoteClosed.Chan:
		err = io.EOF
		return
	}
	select {
	case <-read.proceed:
		msg := read.msg
		n = copy(data, msg.JobSerz)
		if n < len(msg.JobSerz) {
			// buffer the leftover
			s.nextRead = append(s.nextRead, msg.JobSerz[n:]...)
		}
	case <-s.net.halt.ReqStop.Chan:
		err = ErrShutdown
		return
	case timeout := <-readDead:
		_ = timeout
		err = os.ErrDeadlineExceeded
		//err = &simconnError{isTimeout: true, desc: "i/o timeout"}
		return
	case <-s.localClosed.Chan:
		err = io.EOF
		return
	case <-s.remoteClosed.Chan:
		err = io.EOF
		return
	}
	return
}

func (s *simnetConn) Close() error {
	// only close local, might still be bytes to read on other end.
	s.localClosed.Close()
	return nil
}

func (s *simnetConn) LocalAddr() net.Addr {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.local.netAddr
}

func (s *simnetConn) RemoteAddr() net.Addr {
	s.mut.Lock()
	defer s.mut.Unlock()
	return s.remote.netAddr
}

func (s *simnetConn) SetDeadline(t time.Time) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if t.IsZero() {
		s.readDeadlineTimer = nil
		s.sendDeadlineTimer = nil
		return nil
	}
	now := time.Now()
	dur := t.Sub(now)
	s.readDeadlineTimer = s.net.createNewTimer(s.local, dur, now, s.isCli)
	s.sendDeadlineTimer = s.readDeadlineTimer
	return nil
}

func (s *simnetConn) SetWriteDeadline(t time.Time) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if t.IsZero() {
		s.sendDeadlineTimer = nil
		return nil
	}
	now := time.Now()
	dur := t.Sub(now)
	s.sendDeadlineTimer = s.net.createNewTimer(s.local, dur, now, s.isCli)
	return nil
}
func (s *simnetConn) SetReadDeadline(t time.Time) error {
	s.mut.Lock()
	defer s.mut.Unlock()

	if t.IsZero() {
		s.readDeadlineTimer = nil
		return nil
	}
	now := time.Now()
	dur := t.Sub(now)
	s.readDeadlineTimer = s.net.createNewTimer(s.local, dur, now, s.isCli)
	return nil
}

// implements net.Error interface, which
// net.Conn operations return; for Timeout() especially.
type simconnError struct {
	isTimeout bool
	desc      string
}

func (s *simconnError) Error() string {
	return s.desc
}
func (s *simconnError) Timeout() bool {
	return s.isTimeout
}
func (s *simconnError) Temporary() bool {
	return s.isTimeout
}
