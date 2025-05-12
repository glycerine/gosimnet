package gosimnet

// build/run with:
// GOTRACEBACK=all GOEXPERIMENT=synctest go test -v

import (
	"bufio"
	//"context"
	"fmt"
	"net"
	//"os"
	"io"
	//"strings"
	"sync"
	"testing"
	"time"
)

// basic gosimnet operations Listen/Accept, Dial, NewTimer
func Test101_gosimnet_basics(t *testing.T) {

	bubbleOrNot(func() {

		shutdown := make(chan struct{})
		defer close(shutdown)

		cfg := NewSimNetConfig()
		network := NewSimNet(cfg)
		defer network.Close()
		srv := network.NewSimServer("srv_" + t.Name())

		////vv("about to srv.Listen() in %v", t.Name())
		lsn, err := srv.Listen("", "")
		panicOn(err)
		defer srv.Close()
		serverAddr := lsn.Addr()

		// we need the server's conn2 in order
		// to break it out of the Read by conn2.Close()
		// and shutdown all goro cleanly.

		var conn2mut sync.Mutex
		var conn2 []net.Conn
		var done bool
		defer func() {
			conn2mut.Lock()
			defer conn2mut.Unlock()
			done = true
			for _, c := range conn2 {
				c.Close()
			}
		}()

		// make an echo server for the client
		go func() {
			for {
				select {
				case <-shutdown:
					//vv("server exiting on shutdown")
					return
				default:
				}
				c2, err := lsn.Accept()
				if err != nil {
					//vv("server exiting on '%v'", err)
					return
				}
				conn2mut.Lock()
				if done {
					conn2mut.Unlock()
					return
				}
				conn2 = append(conn2, c2)
				conn2mut.Unlock()

				//vv("Accept on conn: local %v <-> %v remote", c2.LocalAddr(), c2.RemoteAddr())
				// per-client connection.
				go func(c2 net.Conn) {
					by := make([]byte, 1000)
					for {
						select {
						case <-shutdown:
							//vv("server conn exiting on shutdown")
							return
						default:
						}
						//vv("server about to read on conn")
						n, err := c2.Read(by)
						if err != nil {
							//vv("server conn exiting on Read error '%v'", err)
							return
						}
						by = by[:n]
						//vv("echo server got '%v'", string(by))
						// must end in \n or client will hang!
						_, err = fmt.Fprintf(c2,
							"hi back from echo server, I saw '%v'\n", string(by))
						if err != nil {
							//vv("server conn exiting on Write error '%v'", err)
							return
						}
						// close the conn to test our EOF sending
						c2.Close()
					} // end for

				}(c2)
			} // end for
		}() // end server

		cli := network.NewSimClient("cli_" + t.Name())
		defer cli.Close()

		//vv("cli about to Dial")
		conn, err := cli.Dial("gosimnet", serverAddr.String())
		//vv("err from Dial() = '%v'", err)
		panicOn(err)
		defer conn.Close()

		fmt.Fprintf(conn, "hello gosimnet")
		response, err := bufio.NewReader(conn).ReadString('\n')
		panicOn(err)
		//vv("client sees response: '%v'", string(response))
		if got, want := string(response), `hi back from echo server, I saw 'hello gosimnet'
`; got != want {
			t.Fatalf("error: want '%v' but got '%v'", want, got)
		}

		// reading more should get EOF, since server now closes the file.
		buf := make([]byte, 1)
		nr, err := conn.Read(buf)
		if err != io.EOF {
			panic(fmt.Sprintf("expected io.EOF, got nr=%v; err = '%v'", nr, err))
		}
		//vv("good: got EOF as we should, since server closes the conn.")

		// timer test
		t0 := time.Now()
		goalWait := time.Second

		// set a timer on the client side.
		timeout := cli.NewTimer(goalWait)
		t1 := <-timeout.C
		_ = t1
		elap := time.Since(t0)

		// must do this, since no automatic GC of gosimnet Timers.
		// note: also no timer Reset at the moment, just Discard
		// and make a new one.
		defer timeout.Discard()

		if elap < goalWait {
			t.Fatalf("timer went off too early! elap(%v) < goalWait(%v)", elap, goalWait)
		}
		//vv("good: finished timer (fired at %v) after %v >= goal %v", t1, elap, goalWait)
	})
}
