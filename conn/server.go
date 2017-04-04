package conn

import (
	"net"
)

// NewServer will return a new server connection
func NewServer(addr string) (sp *Server, err error) {
	var (
		s  Server
		l  net.Listener
		nc net.Conn
	)

	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}

	if nc, err = l.Accept(); err != nil {
		return
	}

	s.c = NewConn(nc)
	sp = &s
	return
}

// Server is a server connection type
type Server struct {
	c *Conn
}

// Get will get a message
func (s *Server) Get(fn func([]byte)) error {
	return s.c.Get(fn)
}

// GetStr will get a message as a string
// Note: This is just a helper utility
func (s *Server) GetStr() (string, error) {
	return s.c.GetStr()
}

// Put will put a message
func (s *Server) Put(b []byte) error {
	return s.c.Put(b)
}
