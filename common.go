package gosimnet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	//"sync/atomic"
	"time"
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
	UserMaxPayload = 1_200_000

	maxMessage = 1_310_720 - 80 // ~ 1 MB max message size, prevents TLS clients from talking to TCP servers, as the random TLS data looks like very big message size. Also lets us test on smaller virtual machines without out-of-memory issues.
)

var ErrTooLong = fmt.Errorf("message message too long: over 1MB; encrypted client vs an un-encrypted server?")

var DebugVerboseCompress bool //= true

// uConn hopefully works for both quic.Stream and net.Conn, universally.
type uConn interface {
	io.Writer
	SetWriteDeadline(t time.Time) error

	io.Reader
	SetReadDeadline(t time.Time) error
}
