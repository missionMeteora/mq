package conn

import (
	"net"
)

// NewClient will return a new client connection
func NewClient(addr string) (cp *Client, err error) {
	var (
		c  Client
		nc net.Conn
	)

	if nc, err = net.Dial("tcp", addr); err != nil {
		return
	}

	c.c = NewConn(nc)
	cp = &c
	return
}

// Client is a client connection type
type Client struct {
	c *Conn
}

// Get will get a message
func (c *Client) Get(fn func([]byte)) error {
	return c.c.Get(fn)
}

// GetStr will get a message as a string
// Note: This is just a helper utility
func (c *Client) GetStr() (string, error) {
	return c.c.GetStr()
}

// Put will put a message
func (c *Client) Put(b []byte) error {
	return c.c.Put(b)
}
