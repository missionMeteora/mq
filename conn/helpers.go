package conn

import "net"

// NewServer will return a new server connection
// Note: This will block until it accepts a connection
func NewServer(addr string) (c *Conn, err error) {
	var (
		l  net.Listener
		nc net.Conn
	)

	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}

	if nc, err = l.Accept(); err != nil {
		return
	}

	c = New()
	if err = c.Connect(nc); err != nil {
		return
	}

	return
}
