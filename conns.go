package mq

import (
	"net"
	"sync"

	"github.com/missionMeteora/jump/chanchan"
)

// newConns returns a pointer to a new instance of conns
func newConns() *conns {
	return &conns{
		m: make(map[Chunk]*conn),
	}
}

type conns struct {
	// TODO (Josh): See about utilizing the R functionality
	mux sync.RWMutex

	// Internal store of connections
	m map[Chunk]*conn
}

// Get will return a conn which matches the provided key. If no match is available, set ok to false
func (c *conns) Get(k Chunk) (out *conn, ok bool) {
	c.mux.Lock()
	out, ok = c.m[k]
	c.mux.Unlock()
	return
}

// Put inserts a conn for the provided key
func (c *conns) Put(k Chunk, nc net.Conn, op Operator, errC *chanchan.ChanChan) (err error) {
	var (
		cc *conn
		ok bool
	)

	c.mux.Lock()
	if cc, ok = c.m[k]; ok && cc.isConnected() {
		// Conn already exists in the list, set err to ErrConnExists
		err = ErrConnExists
	} else {
		// No conn exists for this entry, create a new one
		cc = newConn(k, nil, op, nil, errC)
		// Set new conn as entry for provided key
		c.m[k] = cc
	}

	// At this point, we have a conn. We need to call refreshSettings on it
	err = cc.refreshSettings(k, nc)
	c.mux.Unlock()
	return
}

// Delete removes a connection from the list
func (c *conns) Delete(k Chunk) {
	c.mux.Lock()
	// Remove key from internal storage
	delete(c.m, k)
	c.mux.Unlock()
}

func (c *conns) ForEach(fn func(Chunk, *conn) error) (err error) {
	c.mux.Lock()

	for k, v := range c.m {
		if err = fn(k, v); err != nil {
			break
		}
	}

	c.mux.Unlock()
	return
}

// Returns a list of string keys
func (c *conns) List() (l []Chunk) {
	c.mux.Lock()
	l = make([]Chunk, 0, len(c.m))
	for k := range c.m {
		l = append(l, k)
	}

	c.mux.Unlock()
	return
}
