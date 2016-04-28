package mq

import (
	"net"
	"sync/atomic"

	"github.com/missionMeteora/jump/chanchan"
	jc "github.com/missionMeteora/jump/common"
	"github.com/missionMeteora/mq/internal/common"
)

// NewServer returns a pointer to a new instance of Server
func NewServer(loc string, key Chunk, op Operator, allowed ...KeyToken) (srv *Server, err error) {
	s := Server{
		id:   key,
		a:    newAuth(),
		c:    newConns(),
		op:   op,
		errC: chanchan.NewChanChan(4, 12, chanchan.FullPush),
	}

	for _, kt := range allowed {
		s.PutAuth(kt.Key, kt.Token)
	}

	// Listen at provided location
	if s.l, err = net.Listen("tcp", loc); err != nil {
		// Error encountered while attempting to listen, return err
		return nil, err
	}

	// Start listener loop in a new go routine
	go s.listener()

	return &s, nil
}

// Server is used by a service hosting a mq connection
type Server struct {
	// Listener for inbound TCP connections
	l net.Listener

	id Chunk

	// Auth manager
	a *auth
	// Connections manager
	c *conns
	// Error channel
	errC *chanchan.ChanChan

	// Handshake buffer
	hsBuf [32]byte

	// Operator for handling connection and disconnections
	op Operator

	// Closed state, one represents closed
	closed uint32
}

// Listener processes inbound TCP connections
func (s *Server) listener() {
	var (
		nc  net.Conn
		err error

		// Handshake to be used within the loop
		hs handshake
		ok bool
	)

	// Loop while server is open
	for !s.isClosed() {
		if nc, err = s.l.Accept(); err != nil {
			continue
		}

		if hs, ok = s.handshake(nc); !ok {
			// Invalid message header provided, send a message with a status of Invalid
			sendMsg(nc, common.MTStatement, common.StatusInvalid, nil)
			nc.Close()
			continue
		}

		if !s.a.IsValid(hs) {
			// Credentials are invalid, send a message with a status of Forbidden
			sendMsg(nc, common.MTStatement, common.StatusForbidden, nil)
			nc.Close()
			continue
		}

		if err = s.c.Put(hs.key, nc, s.op, s.errC); err != nil {
			// Error encountered while putting, return error to connecting client
			sendMsg(nc, common.MTStatement, common.StatusError, []byte(err.Error()))
			c.Close()
			continue
		}

		// Connection successful, send server's ID to client
		sendMsg(nc, common.MTStatement, common.StatusOK, s.id[:])
	}
}

func (s *Server) handshake(c net.Conn) (h handshake, ok bool) {
	// Read the handshake using our handshake buffer
	if n, err := c.Read(s.hsBuf[:]); err != nil || n != 32 {
		// Error exists OR handshake length was invalid, return early
		return
	}

	// Return handshake from bytes in the buffer and set ok to true
	return handshake{NewChunk(s.hsBuf[0:16]), NewChunk(s.hsBuf[16:32])}, true
}

func (s *Server) isClosed() bool {
	// Is s.closed set to one? If so, we are closed
	return atomic.LoadUint32(&s.closed) == 1
}

// GetAuth will return the token for a matching key
func (s *Server) GetAuth(key string) (str string, ok bool) {
	var tkn Chunk
	keyC, _ := NewChunkFromString(key)

	// Get token for provided key
	if tkn, ok = s.a.Get(keyC); !ok {
		// No matches exist for this key within the Auth manager
		return
	}

	// Return string representation of our token
	return tkn.String(), ok
}

// PutAuth will put a token as the value for a matching key
func (s *Server) PutAuth(key, token string) (err error) {
	var (
		kC, tC Chunk
	)

	if kC, err = NewChunkFromString(key); err != nil {
		return
	}

	if tC, err = NewChunkFromString(token); err != nil {
		return
	}

	// Convert key and token to Chunks, then call Auth.Put
	s.a.Put(kC, tC)
	return
}

// DeleteAuth will delete the entry matching key (if exists)
func (s *Server) DeleteAuth(key string) {
	// Convert key to Chunk
	kC, _ := NewChunkFromString(key)
	// Call delete on auth
	s.a.Delete(kC)
}

// ListConns returns a list of keys for all the current conns
func (s *Server) ListConns() []Chunk {
	return s.c.List()
}

// Statement is used to send statements to a connection with the provided key
func (s *Server) Statement(key string, b []byte) (err error) {
	var (
		c  *conn
		ok bool
		kC Chunk
	)

	if kC, err = NewChunkFromString(key); err != nil {
		return
	}

	// Get connection
	if c, ok = s.c.Get(kC); !ok {
		// Connection does not exist, return ErrConnDoesNotExist
		return ErrConnDoesNotExist
	}

	// Return any error encountered while calling c.Statement
	return c.Statement(b)
}

// StatementAll is used to send statements to all active connections
func (s *Server) StatementAll(b []byte) error {
	var errs jc.ErrorList
	s.c.ForEach(func(_ Chunk, c *conn) error {
		errs = errs.Append(c.Statement(b))
		return nil
	})

	return errs.Err()
}

// Request is used to send requests to a connection with the provided key
func (s *Server) Request(key string, b []byte, fn ReqFunc) (err error) {
	var (
		c  *conn
		ok bool
		kC Chunk
	)

	if kC, err = NewChunkFromString(key); err != nil {
		return
	}

	if c, ok = s.c.Get(kC); !ok {
		// Connection does not exist, return ErrConnDoesNotExist
		return ErrConnDoesNotExist
	}

	// Return any error encountered while calling c.Request
	return c.Request(b, fn)
}

// RequestAll is used to send statements to all active connections
func (s *Server) RequestAll(b []byte, fn ReqFunc) error {
	var errs jc.ErrorList
	s.c.ForEach(func(_ Chunk, c *conn) error {
		errs = errs.Append(c.Request(b, fn))
		return nil
	})

	return errs.Err()
}

// Receive is used to receive inbound messages from the provided key
func (s *Server) Receive(key string, rec Receiver) (err error) {
	var (
		c  *conn
		ok bool
		kC Chunk
	)

	if kC, err = NewChunkFromString(key); err != nil {
		return
	}

	// Get connection
	if c, ok = s.c.Get(kC); !ok {
		// Connection does not exist, return ErrConnDoesNotExist
		return ErrConnDoesNotExist
	}

	// Return any error encountered while calling c.Receive
	return c.Receive(rec)
}

// IsConnected will return whether or not a client (referenced by key) is connected
func (s *Server) IsConnected(key string) (ok bool) {
	kC, _ := NewChunkFromString(key)
	_, ok = s.c.Get(kC)
	return
}

// Close will close the server and return any error encountered in the process
func (s *Server) Close() error {
	if s == nil {
		return ErrNotInitialized
	}

	if !atomic.CompareAndSwapUint32(&s.closed, 0, 1) {
		// We are already closed, so we can return at this point
		return ErrServerIsClosed
	}

	var errs jc.ErrorList

	// Close listener
	errs = errs.Append(s.l.Close())
	s.c.ForEach(func(_ Chunk, c *conn) (cerr error) {
		errs = errs.Append(c.Close())
		return nil
	})

	return errs.Err()
}
