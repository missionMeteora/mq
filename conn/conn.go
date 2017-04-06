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

// NewConn will return a new connection
func NewConn(nc net.Conn) *Conn {
	var c Conn
	c.nc = nc
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

	onConnect    []func(*Conn) error
	onDisconnect []func(*Conn)

	closed bool
}

// Key will return the generated key for a connection
func (c *Conn) Key() string {
	return c.key.String()
}

// Created will return the created time for a connection
func (c *Conn) Created() time.Time {
	return c.key.Time()
}

// OnConnect will append an OnConnect func
func (c *Conn) OnConnect(fn func(*Conn) error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.onConnect = append(c.onConnect, fn)
}

// OnDisconnect will append an onDisconnect func
func (c *Conn) OnDisconnect(fn func(*Conn)) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.onDisconnect = append(c.onDisconnect, fn)
}

// Get will get a message
func (c *Conn) Get(fn func([]byte)) (err error) {
	var mlen int64
	c.mux.Lock()
	defer c.mux.Unlock()

	if err = binary.Read(c.nc, binary.LittleEndian, &mlen); err != nil {
		return
	}

	if _, err = io.CopyN(c.buf, c.nc, mlen); err != nil {
		return
	}

	if fn != nil {
		fn(c.buf.Bytes())
	}

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

//Put will put a message
func (c *Conn) Put(b []byte) (err error) {
	mlen := int64(len(b))
	c.mux.Lock()
	defer c.mux.Unlock()

	if err = binary.Write(c.nc, binary.LittleEndian, &mlen); err != nil {
		return
	}

	_, err = c.nc.Write(b)
	return
}

// Close will close a connection
func (c *Conn) Close() (err error) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.closed {
		return errors.ErrIsClosed
	}

	err = c.nc.Close()
	c.closed = true

	for _, fn := range c.onDisconnect {
		fn(c)
	}

	return
}

// OnConnectFn is called when a connection occurs
type OnConnectFn func(*Conn) error
