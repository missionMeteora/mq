package mq

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/missionMeteora/iodb"
	"github.com/missionMeteora/jump/chanchan"
	"github.com/missionMeteora/jump/errors"
	"github.com/missionMeteora/jump/uuid"
)

// newConn returns a pointer to a new instance of conn
func newConn(id Chunk, nc net.Conn, op Operator, db *iodb.DB, errC *chanchan.ChanChan) *conn {
	c := conn{
		id: id,
		nc: nc,

		db: db,

		// Inbound message queue with a len of four and a capacity of thirty-two
		in: newMsgQueue(4, 32),
		// Outbound message queue with a len of four and a capacity of thirty-two
		out: newMsgQueue(4, 32),

		rw: newReqWait(),
		pl: newPool(),

		op: op,

		errC: errC,
	}

	//db.CreateBucket("mq", id.String())

	return &c
}

// Conn is the foundation for the mq system. It's role is to coordinate all the needed systems and services to pass messages
type conn struct {
	// Id represented by a [16]byte
	id Chunk

	// TCP connection for conn
	nc net.Conn
	// net.Conn mutex
	ncm sync.Mutex

	db *iodb.DB

	// Inbound message queue
	in *msgQueue
	// Outbound message queue
	out *msgQueue

	// Request func manager
	rw *reqWait
	// Byteslice pool
	pl pool

	// Operator for handling connection and disconnections
	op Operator

	// Error channel
	errC *chanchan.ChanChan

	// Sender sema
	sm sync.Mutex

	// Listener sema
	lm sync.Mutex

	// The current state of the connection:
	//	- Zero represents an ready state
	//	- One represents a connected state
	//	- Two represents a closed state
	state uint32
}

func (c *conn) refreshSettings(id Chunk, nc net.Conn) (err error) {
	c.ncm.Lock()
	if c.isClosed() {
		c.ncm.Unlock()
		return ErrConnIsClosed
	}

	c.out.Close(false)
	c.sm.Lock()
	c.out = newMsgQueue(4, 32)

	// Ensure net.Conn has been closed
	if c.nc != nil {
		c.nc.Close()
	}

	c.lm.Lock()
	c.id = id
	c.nc = nc
	c.ncm.Unlock()
	atomic.SwapUint32(&c.state, 0)

	// Release the locks from the listener and sender semas
	c.lm.Unlock()
	c.sm.Unlock()

	return c.setConnected()
}

func (c *conn) listener() {
	c.lm.Lock()

	var (
		buf  [HeaderLen]byte // Header buffer
		blen int64           // Body length
		n    int             // Amount read
		m    msg             // Message to be used by loop
		err  error           // Error to be used by loop
	)

	// Loop until we encounter an error
	for err == nil {
		// Read from net.Conn into our buffer
		if n, err = c.nc.Read(buf[:]); err != nil {
			// Error was encountered, break
			break
		} else if n != HeaderLen {
			// Read amount does not match HeaderLen (const), message header is invalid
			err = ErrInvalidMsgHeader
			break
		}

		// Message type is located at the twenty-forth index of the buffer
		m.t = msgType(buf[24])

		// Copy index zero to index sixteen (not includeding) to the message id (passed as a slice)
		copy(m.id[:], buf[:16])

		// Set body length by reading eight bytes of the buffer starting at index sixteen
		if blen = *(*int64)(unsafe.Pointer(&buf[16])); blen > 0 {
			// Body length  is greater than zero

			// Set m.Body by getting a slice from the pool for our needed length
			m.body = c.pl.Get(blen)
			// Read from the net connection to m.body
			if n, err = c.nc.Read(m.body); err != nil {
				// An error was encountered, break
				break
			}
		}

		// Process message, if an error is encountered:
		//	- Send message to error chan
		//	- We don't need to kill connection because of an invalid message type, set err to nil
		if err = c.process(m); err != nil {
			c.errC.Send(err)
			err = nil
		}

		// If message body exists, return m.body to pool and set m.body as nil
		if m.body != nil {
			m.body = nil
		}

		// If our connection is closed, set the error to io.EOF
		if !c.isConnected() {
			err = io.EOF
		}
	}

	if err != io.EOF {
		// If we have an error which does not equal io.EOF, send it to the error chan
		c.errC.Send(err)
	}

	c.lm.Unlock()
}

func (c *conn) sender() {
	c.sm.Lock()

	var (
		buf []byte // Message buffer
		n   int    // Message length
		m   msg    // Message to be used by loop
		err error  // Error to be used by loop
	)

	for m, err = c.out.Get(); err == nil; m, err = c.out.Get() {
		// Set buf using slice pool
		buf, n = m.Bytes(c.pl.Get(int64(HeaderLen + len(m.body))))
		// Write buf to net.Conn
		_, err = c.nc.Write(buf[:n])
		// Return buf to slice pool
		c.pl.Put(buf)

		// If our connection is closed, set the error to io.EOF
		if !c.isConnected() {
			err = io.EOF
		}
	}

	if err != io.EOF {
		// If we have an error which does not equal io.EOF, send it to the error chan
		c.errC.Send(err)
	}

	c.sm.Unlock()
}

// Process handles inbound messages and determines which actions need to be taken
func (c *conn) process(m msg) (err error) {
	// Switch on message type
	switch m.t {
	case mtRequest, mtStatement:
		// Put message in inbound queue
		// We do not return body to pool until we are finished using it
		return c.in.Put(m)
	case mtResponse:
		// Get request function for provided message id
		if fn, ok := c.rw.Get(m.id); ok {
			// Call fn with message body as an argument
			fn(m.body)
			// We do not return body to pool until we are finished using it
			return
		}

		// If request function does not exist, set err to ErrReqFnDoesNotExist
		err = ErrReqFnDoesNotExist
	default:
		// Provided type is not any of our currently recognized types, return ErrInvalidmsgType
		err = ErrInvalidmsgType
	}

	if m.body != nil {
		// Body exists, return it to the pool
		c.pl.Put(m.body)
	}

	return
}

func (c *conn) setConnected() error {
	if !atomic.CompareAndSwapUint32(&c.state, 0, 1) {
		// We are already connected, return ErrCannotSetConnected
		return ErrCannotSetConnected
	}

	if c.op != nil {
		// Operator exists, send notification to OnConnect
		c.op.OnConnect(c.id)
	}

	// Start listener loop in a new go routine
	go c.listener()
	// Start sender loop in a new go routine
	go c.sender()
	return nil
}

func (c *conn) isReady() bool {
	return atomic.LoadUint32(&c.state) == 0
}

func (c *conn) isConnected() bool {
	return atomic.LoadUint32(&c.state) == 1
}

func (c *conn) isClosed() bool {
	return atomic.LoadUint32(&c.state) == 2
}

// Statement is a message which does not expect nor accept a response
func (c *conn) Statement(b []byte) (err error) {
	if c.isClosed() {
		return ErrConnIsClosed
	}

	return c.out.Put(msg{uuid.New(), mtStatement, statusOK, b})
}

// Request is a message which expects a response
func (c *conn) Request(b []byte, fn ReqFunc) (err error) {
	if c.isClosed() {
		return ErrConnIsClosed
	}

	m := msg{uuid.New(), mtRequest, statusOK, b}
	c.rw.Put(m.id, fn)
	c.out.Put(m)

	return
}

func (c *conn) Receive(rec Receiver) (err error) {
	if c.isClosed() {
		return ErrConnIsClosed
	}

	var m msg
	// Get next message from inbound queue
	if m, err = c.in.Get(); err != nil {
		return
	}

	// For supported message types:
	// We are going to assume that the end-user is going to hold onto the message body,
	// like a child who is five years old and still carries their Teddy everywhere.
	// As a result, we are going to avoid race-prone situations by saying "Goodbye! Enjoy your new home!"
	// and NOT returning the byteslice to the pool.
	switch m.t {
	case mtRequest:
		err = c.out.Put(msg{
			id:   m.id, // Use same id as requesting message to match on the other side
			t:    mtResponse,
			body: rec.Response(m.body), // Process body and return result to responding body
		})
	case mtStatement:
		rec.Statement(m.body)
	default:
		// This message type is invalid, return message body to pool
		c.pl.Put(m.body)
	}

	return
}

// Close will close the conn and return a list of errors it encounters in the process
func (c *conn) Close() error {
	var errs errors.ErrorList
	if atomic.SwapUint32(&c.state, 2) == 2 {
		// We are already closed, so we can return at this point
		return append(errs, ErrConnIsClosed).Err()
	}

	// Close outbound channel, we are not waiting for close because acquiring c.sm lock will ensure closure
	if err := c.out.Close(false); err != nil {
		// Err exists, append err to errs
		errs = append(errs, err)
	}

	c.sm.Lock()

	// Lock and close net.Conn to avoid additional inbound messages
	c.ncm.Lock()
	if c.nc != nil {
		errs = errs.Append(c.nc.Close())
	}
	c.ncm.Unlock()

	// Close inbound channel, we are not waiting for close because acquiring c.lm lock will ensure closure
	if err := c.in.Close(false); err != nil {
		// Err exists, append err to errs
		errs = append(errs, err)
	}

	c.lm.Lock()

	// Dump remaining waiting funcs
	c.rw.Dump()

	if c.op != nil {
		// Operator exists, send notification to OnDisconnect
		c.op.OnDisconnect(c.id)
	}

	c.lm.Unlock()
	c.sm.Unlock()

	return errs.Err()
}

// readMsg is used as a under the hood helper function utilized during the initialization process
// This is NOT intended to be used once the listener loop begins.
func readMsg(nc net.Conn) (m msg, err error) {
	var (
		buf [HeaderLen]byte // Header buffer
		n   int             // Amount read
	)

	if n, err = io.ReadFull(nc, buf[:]); err != nil {
		// Error was encountered, break
		return
	} else if n != HeaderLen {
		// Read amount does not match HeaderLen (const), message header is invalid
		err = ErrInvalidMsgHeader
		return
	}

	// Message type is located at the twenty-forth index of the buffer
	m.t = msgType(buf[24])

	// Message type is located at the twenty-fifth index of the buffer
	m.s = status(buf[25])

	// Copy index zero to index sixteen (not includeding) to the message id (passed as a slice)
	copy(m.id[:], buf[:16])

	// Save this for later
	//if m.s != statusOK || m.s !=

	// Set body length by reading eight bytes of the buffer starting at index sixteen
	if blen := *(*int64)(unsafe.Pointer(&buf[16])); blen > 0 {
		// Body length  is greater than zero

		// Set m.Body by getting a slice from the pool for our needed length
		// Note: This does not utilize the pool because it's intended to be used
		// very sparsely. Additionally, this is setup as a helper function so that
		// it's not tied to only being used directly by Conns.
		m.body = make([]byte, blen)

		// Read from the net connection to m.body
		if n, err = io.ReadFull(nc, m.body); err != nil {
			// An error was encountered, break
			return
		}
	}

	return
}

// sendMsg is used as a under the hood helper function utilized during the initialization process
// This is NOT intended to be used once the sender loop begins.
func sendMsg(nc net.Conn, t msgType, s status, b []byte) (err error) {
	m := msg{
		t:    t,
		s:    s,
		body: b,
	}

	// Set buf using slice pool
	buf, n := m.Bytes(make([]byte, int64(HeaderLen+len(m.body))))
	// Write buf to net.Conn
	_, err = nc.Write(buf[:n])
	return
}
