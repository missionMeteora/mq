package common

import (
	"sync"
	"time"
)

// MsgType represents the type of message being sent over the wire
type MsgType uint8

const (
	// MTRequest represents a request message
	MTRequest MsgType = iota
	// MTResponse represents a response message
	MTResponse
	// MTStatement represents a statement message
	MTStatement
)

// Status indicates the status of an inbound message.
// Note: Messages with StatusOK or StatusError are the only messages which expect a body
type Status uint8

const (
	// StatusOK represents an ok message (default)
	StatusOK Status = iota
	// StatusPing is for ping requests
	StatusPing
	// StatusPong is for pong responses
	StatusPong
	// StatusError represents an error message (may contain body)
	StatusError
	// StatusForbidden notifies a client that access to server is not permitted (check key/token pair)
	StatusForbidden
	// StatusInvalid will be returned when a provided message has an invalid header
	StatusInvalid
	// StatusDupConn is returned when the provided key is already connected to a server
	StatusDupConn
)

const (
	// HeaderLen is the static length of message headers, it consists of:
	// - UUID: 16 bytes
	// - Body len: 8 bytes
	// - Message type: 1 byte
	// - Message status: 1 byte
	HeaderLen = 26
)

// NewDialback returns a pointer to a new instance of Dialback
func NewDialback(cap, mult int) *Dialback {
	return &Dialback{
		cap:  cap,
		mult: mult,
	}
}

// Dialback is used to manage dial-back timing
type Dialback struct {
	mux  sync.Mutex
	n    int
	cap  int
	mult int
}

// Next will return the number of seconds until the next dial
func (d *Dialback) Next() (seconds int) {
	d.mux.Lock()
	if d.n < d.cap {
		d.n++
	}

	seconds = d.n * d.mult
	d.mux.Unlock()
	return
}

// Reset will reset the dialback
func (d *Dialback) Reset() {
	d.mux.Lock()
	d.n = 0
	d.mux.Unlock()
}

// Wait will wait until the next dialback time
func (d *Dialback) Wait() {
	s := d.Next()
	time.Sleep(time.Second * time.Duration(s))
}

// NewSema returns a pointer to a new semaphore
func NewSema() *Sema {
	return &Sema{
		c: make(chan struct{}, 1),
	}
}

// Sema is a simple semaphore helper
type Sema struct {
	c chan struct{}
}

// Acquire will acquire a lock from the semaphore
func (s *Sema) Acquire() {
	s.c <- struct{}{}
}

// Release will release a lock from the semaphore
func (s *Sema) Release() {
	select {
	case <-s.c:
	default:
		panic("tried to release a sema which has not been acquired")
	}
	return
}
