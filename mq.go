package mq

import (
	"errors"

	"github.com/missionMeteora/jump/chanchan"
)

var (
	// ErrConnExists is returned when a connection with the provided key already exists
	ErrConnExists = errors.New("connection with this key already exists")

	// ErrConnDoesNotExist is returned when a connection with a requested key does not exist
	ErrConnDoesNotExist = errors.New("connection with this key does not exist")

	// ErrServerIsClosed is returned when an action is attempted on a closed server
	ErrServerIsClosed = errors.New("cannot perform action on a closed server")

	// ErrClientIsClosed is returned when an action is attempted on a closed client
	ErrClientIsClosed = errors.New("cannot perform action on a closed client")

	// ErrConnIsClosed is returned when an action is attempted on a closed connection
	ErrConnIsClosed = errors.New("cannot perform action on a closed connection")

	// ErrInvalidmsgType is returned when an invalid message type is provided
	ErrInvalidmsgType = errors.New("invalid message type provided")

	// ErrInvalidMsgHeader is returned when an invalid message header is provided
	ErrInvalidMsgHeader = errors.New("invalid message header provided")

	// ErrInvalidMsgLength is returned when a message has an invalid length
	ErrInvalidMsgLength = errors.New("invalid message length")

	// ErrReqFnDoesNotExist is returned when a request function does not exist
	ErrReqFnDoesNotExist = errors.New("request function does not exist")

	// ErrCannotSetConnected is returned when a non-open connection is attempted to be set as connected
	ErrCannotSetConnected = errors.New("cannot set to connected, connection is already connected or closed")

	// ErrInvalidstatus is returned when an invalid status is provided in a message by a connected node
	ErrInvalidstatus = errors.New("invalid message status")

	// ErrForbidden is returned when a connecting node does not have access to a requested server
	ErrForbidden = errors.New("forbidden")

	// ErrNotInitialized is returned when an action is attempted on a connection which has not yet been initialized
	ErrNotInitialized = errors.New("cannot perform action on a non-initialized connection")

	// ErrCannotReplaceActiveNetConn is returned when a net.Conn is attempted to be replaced while it is still active
	ErrCannotReplaceActiveNetConn = errors.New("cannot replace active net.Conn")

	// ErrInvalidChunkLen is returned when a string provided is too long to be converted into a chunk
	ErrInvalidChunkLen = errors.New("chunk length cannot exceed 16 characters")
)

// ReqFunc is used when receiving a response or a statement
type ReqFunc func([]byte)

// RespFunc is used when responding to a request
type RespFunc func([]byte) []byte

// Receiver is used to respond to inbound messages
type Receiver interface {
	// Inbound message expects a response
	Response([]byte) []byte
	// Inbound message is not expecting a response
	Statement([]byte)
}

// NewRec returns a pointer to a new Rec
func NewRec(res func([]byte) []byte, stmnt func([]byte)) *Rec {
	return &Rec{res, stmnt}
}

// Rec is a public pre-defined Receiver
type Rec struct {
	res   func([]byte) []byte
	stmnt func([]byte)
}

// Response is a func for responses
func (r *Rec) Response(b []byte) []byte {
	if r.res == nil {
		return nil
	}

	return r.res(b)
}

// Statement is a func for statements
func (r *Rec) Statement(b []byte) {
	if r.stmnt == nil {
		return
	}

	r.stmnt(b)
}

// Operator is used in the creation of both Servers and Clients
type Operator interface {
	// Called when connection is opened. The ID of the connecting node will be passed
	OnConnect(Chunk) error
	// Called when connection is closed. The ID of the disconnected node will be passed
	OnDisconnect(Chunk)
}

// NewOp returns a pointer to a new Op
func NewOp(onC func(Chunk) error, onDC func(Chunk)) *Op {
	return &Op{onC, onDC}
}

// Op is a public pre-defined Operator
type Op struct {
	onC  func(Chunk) error
	onDC func(Chunk)
}

// OnConnect will call onC when a node disconnects
func (o *Op) OnConnect(c Chunk) error {
	if o.onC == nil {
		// onC does not exist, return early
		return nil
	}

	return o.onC(c)
}

// OnDisconnect will call onDC when a node disconnects
func (o *Op) OnDisconnect(c Chunk) {
	if o.onDC == nil {
		// onDC does not exist, return early
		return
	}

	o.onDC(c)
}

// handshake is used to determine if a connecting client is a match for the server's auth list
type handshake struct {
	key   Chunk
	token Chunk
}

// NewChunk returns a new chunk using the provided byteslice
func NewChunk(b []byte) (c Chunk, err error) {
	if len(b) > 16 {
		// We are going to set error, then continue on. The reason for this is
		// because some users may actively know their strings are being truncated
		// and do not care.
		err = ErrInvalidChunkLen
	}
	copy(c[:], b)
	return
}

// NewChunkFromString returns a new chunk using the provided byteslice
func NewChunkFromString(s string) (c Chunk, err error) {
	if len(s) > 16 {
		// We are going to set error, then continue on. The reason for this is
		// because some users may actively know their strings are being truncated
		// and do not care.
		err = ErrInvalidChunkLen
	}
	copy(c[:], s)
	return
}

// Chunk is a 16 byte array with helper functions
type Chunk [16]byte

// String returns the Chunk as a string
func (c Chunk) String() string {
	for i, v := range c {
		if v == 0 {
			return string(c[:i])
		}
	}

	return string(c[:])
}

func newMsgQueue(len, cap int) *msgQueue {
	return &msgQueue{
		cc: chanchan.NewChanChan(len, cap, chanchan.FullWait),
	}
}

// msgQueue holds messages waiting to be processed
type msgQueue struct {
	cc *chanchan.ChanChan
}

// Get returns the next message in the queue
func (mq *msgQueue) Get() (m msg, err error) {
	var v interface{}
	if v, err = mq.cc.Receive(true); err == nil {
		m = v.(msg)
	}

	return
}

// Put adds a message to the queue
func (mq *msgQueue) Put(m msg) error {
	return mq.cc.Send(m)
}

// Close will close the internal chanchan and return any error encountered while closing
func (mq *msgQueue) Close(wait bool) error {
	return mq.cc.Close(wait)
}

// KeyToken is used to pass the key and token as a pair in string format
type KeyToken struct {
	Key   string
	Token string
}
