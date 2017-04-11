package conn

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/missionMeteora/toolkit/errors"
	"github.com/missionMeteora/uuid"
)

const (
	// ErrCannotConnect is returned when a connection is not idle when Connect() is called
	ErrCannotConnect = errors.Error("cannot connect to connected or closed connections")
	// ErrIsIdle is returned when an action is attempted on an idle connection
	ErrIsIdle = errors.Error("cannot perform action on idle connection")
)

const (
	stateIdle uint8 = iota
	stateConnected
	stateClosed
)

// New will return a new connection
func New() *Conn {
	var c Conn
	c.buf = bytes.NewBuffer(nil)
	c.key = uuid.New()
	return &c
}

// Conn is a connection
type Conn struct {
	mux sync.RWMutex
	nc  net.Conn
	buf *bytes.Buffer

	key uuid.UUID

	// On connect functions
	onC []OnConnectFn
	// On disconnect functions
	onDC []OnDisconnectFn

	state uint8
}

func (c *Conn) onConnect() (err error) {
	for _, fn := range c.onC {
		if err = fn(c); err != nil {
			return
		}
	}

	return
}

func (c *Conn) onDisconnect() (err error) {
	for _, fn := range c.onDC {
		fn(c)
	}

	return
}

// get is a raw internal call for getting a message, does not handle locking nor post-get cleanup
func (c *Conn) get(fn func([]byte)) (err error) {
	// Let's ensure our connection is not closed or idle
	switch c.state {
	case stateClosed:
		return errors.ErrIsClosed
	case stateIdle:
		return ErrIsIdle
	}

	var mlen int64
	// Read message length
	if err = binary.Read(c.nc, binary.LittleEndian, &mlen); err != nil {
		return
	}

	// Read message
	if _, err = io.CopyN(c.buf, c.nc, mlen); err != nil {
		return
	}

	if fn != nil {
		// Please do not use the bytes outside of the called functions\
		// I'll be a sad panda if you create a race condition
		fn(c.buf.Bytes())
	}

	return
}

// put is the raw internal call for sending a message, does not handle locking
func (c *Conn) put(b []byte) (err error) {
	// Let's ensure our connection is not closed or idle
	switch c.state {
	case stateClosed:
		return errors.ErrIsClosed
	case stateIdle:
		return ErrIsIdle
	}

	// Set message length
	mlen := int64(len(b))

	// Write the message length
	if err = binary.Write(c.nc, binary.LittleEndian, &mlen); err != nil {
		return
	}

	// Write message
	_, err = c.nc.Write(b)
	return
}

// close will set the state to closed
func (c *Conn) close() (err error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.state == stateClosed {
		return errors.ErrIsClosed
	}

	c.state = stateClosed
	return
}

func (c *Conn) setIdle() {
	if c.state != stateIdle {
		if c.nc != nil {
			c.nc.Close()
		}

		c.nc = nil
		c.state = stateIdle
	}
}

func (c *Conn) setConnection(nc net.Conn) (err error) {
	c.mux.Lock()
	if c.state != stateIdle {
		err = ErrCannotConnect
	} else {
		c.nc = nc
		c.state = stateConnected
	}
	c.mux.Unlock()
	return
}

// Connect will connect a connection to a net.Conn
func (c *Conn) Connect(nc net.Conn) (err error) {
	if err = c.setConnection(nc); err != nil {
		return
	}

	if err = c.onConnect(); err != nil {
		c.mux.Lock()
		c.setIdle()
		c.mux.Unlock()
	}

	return
}

// Key will return the generated key for a connection
func (c *Conn) Key() string {
	// We don't need to lock because the key never changes after creation
	return c.key.String()
}

// Created will return the created time for a connection
func (c *Conn) Created() time.Time {
	// We don't need to lock because the key never changes after creation
	return c.key.Time()
}

// OnConnect will append an OnConnect func
// Note: The referenced conn is returned for chaining
func (c *Conn) OnConnect(fns ...OnConnectFn) *Conn {
	c.mux.Lock()
	c.onC = append(c.onC, fns...)
	c.mux.Unlock()
	return c
}

// OnDisconnect will append an onDisconnect func
// Note: The referenced conn is returned for chaining
func (c *Conn) OnDisconnect(fns ...OnDisconnectFn) *Conn {
	c.mux.Lock()
	c.onDC = append(c.onDC, fns...)
	c.mux.Unlock()
	return c
}

// Get will get a message
// Note: If fn is nil, the message will be read and discarded
func (c *Conn) Get(fn func([]byte)) (err error) {
	c.mux.Lock()
	if err = c.get(fn); err != nil {
		c.setIdle()
	}
	c.mux.Unlock()
	c.buf.Reset()
	return
}

// GetStr will get a message as a string
// Note: This is just a helper utility
func (c *Conn) GetStr() (msg string, err error) {
	err = c.Get(func(b []byte) {
		msg = string(b)
	})

	return
}

// Put will put a message
func (c *Conn) Put(b []byte) (err error) {
	c.mux.Lock()
	err = c.put(b)
	c.mux.Unlock()
	return
}

// Close will close a connection
func (c *Conn) Close() (err error) {
	if err = c.close(); err != nil {
		// Connection is already closed, return early
		return
	}

	// Call onDisconnect before we close the net.Conn
	c.onDisconnect()
	// Close net.Conn
	err = c.nc.Close()
	return
}

// OnConnectFn is called when a connection occurs
type OnConnectFn func(*Conn) error

// OnDisconnectFn is called when a connection ends
type OnDisconnectFn func(*Conn)
