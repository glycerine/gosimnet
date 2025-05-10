package gosimnet

// build/run with:
// GOTRACEBACK=all GOEXPERIMENT=synctest go test -v

import (
	"bufio"
	//"context"
	"fmt"
	"net"
	//"os"
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

		network := NewNet()
		defer network.Close()
		srv := network.NewServer("srv_" + t.Name())

		vv("about to srv.Listen() in %v", t.Name())
		serverAddr, err := srv.Listen()
		panicOn(err)
		defer srv.Close()

		// we need the server's conn2 in order
		// to break it out of the Read by conn2.Close()
		// and shutdown all goro cleanly.

		var conn2mut sync.Mutex
		var conn2 []net.Conn
		defer func() {
			conn2mut.Lock()
			defer conn2mut.Unlock()
			for _, c := range conn2 {
				c.Close()
			}
		}()

		// make an echo server for the client
		go func() {
			for {
				select {
				case <-shutdown:
					vv("server exiting on shutdown")
					return
				default:
				}
				c2, err := srv.Accept()
				if err != nil {
					vv("server exiting on '%v'", err)
					return
				}
				conn2mut.Lock()
				conn2 = append(conn2, c2)
				conn2mut.Unlock()

				vv("Accept on conn: local %v <-> %v remote", c2.LocalAddr(), c2.RemoteAddr())
				// per-client connection.
				go func(c2 net.Conn) {
					by := make([]byte, 1000)
					for {
						select {
						case <-shutdown:
							vv("server conn exiting on shutdown")
							return
						default:
						}
						vv("server about to read on conn")
						n, err := c2.Read(by)
						if err != nil {
							vv("server conn exiting on Read error '%v'", err)
							return
						}
						by = by[:n]
						vv("echo server got '%v'", string(by))
						// must end in \n or client will hang!
						_, err = fmt.Fprintf(c2,
							"hi back from echo server, I saw '%v'\n", string(by))
						if err != nil {
							vv("server conn exiting on Write error '%v'", err)
							return
						}
					} // end for

				}(c2)
			} // end for
		}() // end server

		cli := network.NewClient("cli_" + t.Name())
		defer cli.Close()

		vv("cli about to Dial")
		conn, err := cli.Dial("gosimnet", serverAddr.String())
		vv("err = '%v'", err) // simnet_test.go:82 2000-01-01 00:00:00.002 +0000 UTC err = 'this client is already connected. create a NewClient()'
		panicOn(err)
		defer conn.Close()

		fmt.Fprintf(conn, "hello gosimnet")
		response, err := bufio.NewReader(conn).ReadString('\n')
		panicOn(err)
		vv("client sees response: '%v'", string(response)) // not seen

		// timer test
		t0 := time.Now()
		goalWait := time.Second

		// set a timer on the client side.
		timeout := cli.NewTimer(goalWait)
		t1 := <-timeout.C
		elap := time.Since(t0)

		// must do this, since no automatic GC of gosimnet Timers.
		// note: also no timer Reset at the moment, just Discard
		// and make a new one.
		defer timeout.Discard()

		if elap < goalWait {
			t.Fatalf("timer went off too early! elap(%v) < goalWait(%v)", elap, goalWait)
		}
		vv("good: finished timer (fired at %v) after %v >= goal %v", t1, elap, goalWait)
	})
}
