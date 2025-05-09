package gosimnet

import (
	"bytes"
	//"encoding/hex"
	"encoding/json"
	//"os"
	//"sync/atomic"
	"time"

	"context"
	"fmt"
	"io"
	"net"
	"sync"

	//cryrand "crypto/rand"
	//cristalbase64 "github.com/cristalhq/base64"
	//"github.com/glycerine/greenpack/msgp"
	"github.com/glycerine/idem"
	//"github.com/glycerine/loquet"
	gjson "github.com/goccy/go-json"
)

const (
	UserMaxPayload = 1_200_000 // users should chunk to this size, to be safe.
)

var _ = io.EOF
var ErrShutdown = fmt.Errorf("shutting down")

// uConn hopefully works for both quic.Stream and net.Conn, universally.
type uConn interface {
	io.Writer
	SetWriteDeadline(t time.Time) error

	io.Reader
	SetReadDeadline(t time.Time) error
}

// stubs

type Client struct {
	mut     sync.Mutex
	cfg     *Config
	name    string
	halt    *idem.Halter
	simnode *simnode
	simnet  *simnet

	simconn *simnetConn
	conn    *simnetConn

	connected chan error
}

func NewClient() *Client {
	return &Client{
		halt:      idem.NewHalter(),
		connected: make(chan error, 1),
	}
}

func NewServer() *Server {
	return &Server{
		halt: idem.NewHalter(),
		//connected: make(chan error, 1),
	}
}

type Server struct {
	mut                sync.Mutex
	cfg                *Config
	name               string
	halt               *idem.Halter
	simnode            *simnode
	simnet             *simnet
	boundAddressString string
}

type Config struct {
	simnetRendezvous *simnetRendezvous
	localAddress     string

	ClientDialToHostPort string
}

func (c *Client) setLocalAddr(conn localRemoteAddr) {
	c.mut.Lock()
	defer c.mut.Unlock()

	c.cfg.localAddress = local(conn)
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

func NewConfig() *Config {
	return &Config{
		simnetRendezvous: &simnetRendezvous{},
	}
}

// gotta have just one simnet, shared by all
// clients and servers for a single test/Configure.
type simnetRendezvous struct {
	singleSimnetMut sync.Mutex
	singleSimnet    *simnet
}

type Message struct {

	// HDR contains header information.
	HDR HDR `zid:"0"`

	// JobSerz is the "body" of the message.
	// The user provides and interprets this.
	JobSerz []byte `zid:"1"`

	// JobErrs returns error information from the server-registered
	// user-defined callback functions.
	JobErrs string `zid:"2"`

	// LocalErr is not serialized on the wire by the server.
	// It communicates only local (client/server side) information.
	//
	// Callback functions convey
	// errors in JobErrs (by returning an error);
	// or in-band within JobSerz.
	LocalErr error `msg:"-"`

	// DoneCh.WhenClosed will be closed on the client when the one-way is
	// sent or the round-trip call completes.
	// NewMessage() automatically allocates DoneCh correctly and
	// should be used when creating a new Message (on the client to send).
	//DoneCh *loquet.Chan[Message] `msg:"-"`

	nextOrReply *Message // free list on server, replies to round-trips in the client.
}

// CopyForSimNetSend is used by the simnet
// so that senders can overwrite their sent
// messages once they are "in the network", emulating
// the copy that the kernel does for socket writes.
// For safety, this cloning zeroes DoneCh, nextOrReply, ...
// to avoid false-sharing; anything marked `msg:"-"`
// would not be serialized by greenpack when
// sent over a network.
// In cl.HDR, we nil out Nc, Ctx, and streamCh
// -- they are marked `msg:"-"` and are expected
// to be set by the receiver when needed.
func (m *Message) CopyForSimNetSend() (c *Message) {
	cp := *m
	c = &cp
	c.nextOrReply = nil
	// make our own copy of these central/critical bytes.
	c.JobSerz = append([]byte{}, m.JobSerz...)
	c.LocalErr = nil // marked msg:"-"
	// like MessageFromGreenpack, DoneCh is
	// not needed for sends/reads.
	//c.DoneCh = nil
	c.nextOrReply = nil

	c.HDR.Nc = nil
	c.HDR.Args = make(map[string]string)
	for k, v := range m.HDR.Args {
		c.HDR.Args[k] = v
	}
	c.HDR.Ctx = nil
	c.HDR.streamCh = nil
	return
}

// interface for goq

// NewMessage allocates a new Message with a DoneCh properly created.
func NewMessage() *Message {
	m := &Message{}
	//m.DoneCh = loquet.NewChan(m)
	m.HDR.Args = make(map[string]string)
	return m
}

// String returns a string representation of msg.
func (msg *Message) String() string {
	return fmt.Sprintf("&Message{HDR:%v, LocalErr:'%v', len %v JobSerz}", msg.HDR.String(), msg.LocalErr, len(msg.JobSerz))
}

// NewMessageFromBytes calls NewMessage() and sets by as the JobSerz field.
func NewMessageFromBytes(by []byte) (msg *Message) {
	msg = NewMessage()
	msg.JobSerz = by
	return
}

type HDR struct {

	// Nc is supplied to reveal the LocalAddr() or RemoteAddr() end points.
	// Do not read from, or write to, this connection;
	// that will cause the RPC connection to fail.
	Nc net.Conn `msg:"-"`

	Created time.Time `zid:"0"` // HDR creation time stamp.
	From    string    `zid:"1"` // originator host:port address.
	To      string    `zid:"2"` // destination host:port address.

	ServiceName string `zid:"11"` // registered name to call.

	// arguments/parameters for the call. should be short to keep the HDR small.
	// big stuff should be serialized in JobSerz.
	Args map[string]string `zid:"12"`

	Subject string `zid:"3"` // in net/rpc, the "Service.Method" ServiceName
	Seqno   uint64 `zid:"4"` // user (client) set sequence number for each call (same on response).
	Typ     int    `zid:"5"` // see constants above.
	CallID  string `zid:"6"` // 20 bytes pseudo random base-64 coded string (same on response).
	Serial  int64  `zid:"7"` // system serial number

	LocalRecvTm time.Time `zid:"8"`

	// allow standard []byte oriented message to cancel too.
	Ctx context.Context `msg:"-"`

	// Deadline is optional, but if it is set on the client,
	// the server side context.Context will honor it.
	Deadline time.Time `zid:"9"` // if non-zero, set this deadline in the remote Ctx

	// The CallID will be identical on
	// all parts of the same stream.
	StreamPart int64 `zid:"10"`

	// NoSystemCompression turns off any usual
	// compression that the rpc25519 system
	// applies, for just sending this one Message.
	//
	// Not normally a needed (or a good idea),
	// this flag is for efficiency when the
	// user has implemented their own custom compression
	// scheme for the JobSerz data payload.
	//
	// By checking this flag, the system can
	// avoid wasting time attempting
	// to compress a second time; since the
	// user has, hereby, marked this Message
	// as incompressible.
	//
	// Not matched in reply compression;
	// this flag will not affect the usual
	// compression-matching in responses.
	// For those purposes, it is ignored.
	NoSystemCompression bool `zid:"13"`

	// ToPeerID and FromPeerID help maintain stateful sub-calls
	// allowing client/server symmetry when
	// implementing complex stateful protocols.
	ToPeerID   string `zid:"14"`
	FromPeerID string `zid:"15"`
	FragOp     int    `zid:"16"`

	// streamCh is internal; used for client -> server streaming on CallUploadBegin
	streamCh chan *Message `msg:"-" json:"-"`
}

func (m *HDR) String() string {
	//return m.Pretty()
	return fmt.Sprintf(`&rpc25519.HDR{
    "Created": %v,
    "From": %v,
    "To": %v,
    "ServiceName": %v,
    "Args": %#v,
    "Subject": %v,
    "Seqno": %v,
    "Typ": %v,
    "CallID": %v,
    "Serial": %v,
    "LocalRecvTm": "%v",
    "Deadline": "%v",
    "FromPeerID": "%v",
    "ToPeerID": "%v",
    "StreamPart": %v,
    "FragOp": %v,
}`,
		m.Created,
		m.From,
		m.To,
		m.ServiceName,
		m.Args,
		m.Subject,
		m.Seqno,
		m.Typ,
		m.CallID,
		m.Serial,
		m.LocalRecvTm,
		m.Deadline,
		m.FromPeerID,
		m.ToPeerID,
		m.StreamPart,
		m.FragOp,
	)
}

// Compact is all on one line.
func (m *HDR) Compact() string {
	return fmt.Sprintf("%#v", m)
}

// JSON serializes to JSON.
func (m *HDR) JSON() []byte {
	jsonData, err := json.Marshal(m)
	panicOn(err)
	return jsonData
}

// Bytes serializes to compact JSON formatted bytes.
func (m *HDR) Bytes() []byte {
	return m.JSON()
}

// Unbytes reverses Bytes.
func Unbytes(jsonData []byte) *HDR {
	var mid HDR
	err := gjson.Unmarshal(jsonData, &mid)
	panicOn(err)
	return &mid
}

func HDRFromBytes(jsonData []byte) (*HDR, error) {
	var mid HDR
	err := gjson.Unmarshal(jsonData, &mid)
	if err != nil {
		return nil, err
	}
	return &mid, nil
}

// Pretty shows in pretty-printed JSON format.
func (m *HDR) Pretty() string {
	by := m.JSON()
	var pretty bytes.Buffer
	err := json.Indent(&pretty, by, "", "    ")
	panicOn(err)
	return pretty.String()
}

type localRemoteAddr interface {
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

type uConnLR interface {
	uConn
	localRemoteAddr
}
