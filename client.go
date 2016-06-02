package mq

import (
	"net"
	"sync/atomic"

	"github.com/missionMeteora/jump/chanchan"
	"github.com/missionMeteora/jump/errors"
)

// NewClient returns a pointer to a new instance of Client
func NewClient(opts ClientOpts) (cl *Client, err error) {
	// Initialize new client and connection
	cl = &Client{
		dialb: Newdialback(6, 5),
		loc:   opts.Loc,
		op:    opts.Op,
		errC:  chanchan.NewChanChan(4, 12, chanchan.FullPush),
	}

	if cl.key, err = NewChunkFromString(opts.Name); err != nil {
		return
	}

	if cl.token, err = NewChunkFromString(opts.Token); err != nil {
		return
	}

	if cl.op == nil {
		cl.op = NewOp(nil, nil)
	}

	cl.conn = newConn(Chunk{}, nil, NewOp(cl.op.OnConnect, cl.onDisconnect), nil, cl.errC)

	// Dial within a goroutine so that we don't hold up the initalization process
	go func() {
		if err := cl.Dial(); err != nil {
			cl.errC.Send(err)
			return
		}
	}()
	return
}

// Client is used to connect to Server instances
type Client struct {
	*conn

	dialb *dialback

	loc   string
	key   Chunk
	token Chunk
	op    Operator

	// Internal full-access channel
	errC *chanchan.ChanChan

	closed uint32
}

func (c *Client) onDisconnect(key Chunk) {
	if c.isClosed() {
		return
	}

	go c.op.OnDisconnect(key)

	c.dialb.Wait()
	c.Dial()
}

// Dial will connect to the server
func (c *Client) Dial() (err error) {
	var (
		nc net.Conn
		id Chunk
	)

	for nc, err = net.Dial("tcp", c.loc); err != nil; nc, err = net.Dial("tcp", c.loc) {
		// Waiting, then attempting to reconnect
		c.dialb.Wait()
	}

	if id, err = clientHandshake(nc, c.key, c.token); err != nil {
		// We encountered an error writing our handshake to the server.
		return
	}

	// Set new net.Conn
	c.refreshSettings(id, nc)

	// Dialed, shook hands, and toasted glasses. We can now set our status as "connected"
	c.setConnected()

	// Reset the dialback
	c.dialb.Reset()
	return
}

// ErrC returns a chanchan.Receiver interface which is backed by c.errC
func (c *Client) ErrC() chanchan.Receiver {
	return c.errC
}

// Close will close the client instance
func (c *Client) Close() error {
	if c == nil {
		return ErrNotInitialized
	}

	if !atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		// We are already closed, so we can return at this point
		return ErrClientIsClosed
	}

	var errs errors.ErrorList
	// Close our internal connection
	if err := c.conn.Close(); err != nil {
		errs = errs.Append(err)
	}

	// Close our error channel
	if err := c.errC.Close(false); err != nil {
		errs = errs.Append(err)
	}

	return errs.Err()
}

// clientHandshake will use a key and token to send a handshake to the server
func clientHandshake(nc net.Conn, key, token Chunk) (id Chunk, err error) {
	var hs [64]byte
	// Copy key to the first sixteen bytes
	copy(hs[:32], key[:])
	// Copy token to the last sixteen bytes
	copy(hs[32:], token[:])
	// Send handhshake to server
	if _, err = nc.Write(hs[:]); err != nil {
		return
	}

	var m msg
	// Read handshake message
	if m, err = readMsg(nc); err != nil {
		return
	}

	// Switch on handshake status
	switch m.s {
	case statusInvalid:
		err = ErrInvalidMsgHeader
	case statusForbidden:
		err = ErrForbidden
	case statusOK:
		// Everything is good, move along now
	default:
		// None of our expected statuses were provided, set err to ErrInvalidstatus
		err = ErrInvalidstatus
	}

	if err == nil {
		// We have no errors, get id from message body and return it!
		id, err = NewChunk(m.body)
	}

	return
}
