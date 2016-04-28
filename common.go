package mq

import (
	"sync"
	"time"
)

// msgType represents the type of message being sent over the wire
type msgType uint8

const (
	// mtRequest represents a request message
	mtRequest msgType = iota
	// mtResponse represents a response message
	mtResponse
	// mtStatement represents a statement message
	mtStatement
)

// status indicates the status of an inbound message.
// Note: Messages with statusOK or statusError are the only messages which expect a body
type status uint8

const (
	// statusOK represents an ok message (default)
	statusOK status = iota
	// statusPing is for ping requests
	statusPing
	// statusPong is for pong responses
	statusPong
	// statusError represents an error message (may contain body)
	statusError
	// statusForbidden notifies a client that access to server is not permitted (check key/token pair)
	statusForbidden
	// statusInvalid will be returned when a provided message has an invalid header
	statusInvalid
	// statusDupConn is returned when the provided key is already connected to a server
	statusDupConn
)

const (
	// HeaderLen is the static length of message headers, it consists of:
	// - UUID: 16 bytes
	// - Body len: 8 bytes
	// - Message type: 1 byte
	// - Message status: 1 byte
	HeaderLen = 26
)

// Newdialback returns a pointer to a new instance of dialback
func Newdialback(c, m int) *dialback {
	return &dialback{
		c: c,
		m: m,
	}
}

// dialback is used to manage dial-back timing
type dialback struct {
	mux sync.Mutex
	// Current count
	n int
	// Capacity
	c int
	// Multiplier
	m int
}

// Next will return the number of seconds until the next dial
func (d *dialback) Next() (seconds int) {
	d.mux.Lock()
	if d.n < d.c {
		d.n++
	}

	seconds = d.n * d.m
	d.mux.Unlock()
	return
}

// Reset will reset the dialback
func (d *dialback) Reset() {
	d.mux.Lock()
	d.n = 0
	d.mux.Unlock()
}

// Wait will wait until the next dialback time
func (d *dialback) Wait() {
	s := d.Next()
	time.Sleep(time.Second * time.Duration(s))
}
