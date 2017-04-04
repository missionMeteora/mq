package pubsub

import (
	"net"
	"sync"

	"github.com/missionMeteora/journaler"
	"github.com/missionMeteora/mq/conn"
	"github.com/missionMeteora/uuid"
)

// NewPub will return a new publisher
func NewPub(addr string) (pp *Pub, err error) {
	var p Pub
	p.sm = make(map[string]*conn.Conn)
	p.out = journaler.New("Pub", addr)

	var l net.Listener
	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}

	go p.listen(l)
	pp = &p
	return
}

// Pub is a publisher
type Pub struct {
	mux sync.RWMutex
	sm  map[string]*conn.Conn
	out *journaler.Journaler
}

// Listen will listen
func (p *Pub) listen(l net.Listener) {
	var err error

	if l, err = net.Listen("tcp", addr); err != nil {
		return
	}

	for {
		var nc net.Conn
		if nc, err = l.Accept(); err != nil {
			p.out.Error(err)
			continue
		}

		c := conn.NewConn(nc)
		c.OnDisconnect(p.remove)

		p.mux.Lock()
		p.sm[c.Key()] = c
		p.mux.Unlock()
	}
}

// Put will broadcast a message to all subscribers
func (p *Pub) Put(b []byte) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	for _, c := range p.sm {
		c.Put(b)
	}
}

// Subscribers will provide a map of subscribers with their creation time as the value
func (p *Pub) Subscribers() (sm map[string]time.Time) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	sm = make(map[string]time.Time, len(p.sm))
	for _, c := range p.sm {
		sm[c.Key()] = c.Created()
	}

	return
}

func (p *Pub) get(key string) (c *conn.Conn, ok bool) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	c, ok = p.sm[key]
	return
}

// Remove will remove a subscriber
func (p *Pub) Remove(key string) (err error) {
	p.mux.RLock()
	defer p.mux.RUnlock()

	if c, ok := p.get(key); ok {
		err = c.Close()
	}

	return
}

func (p *Pub) remove(c *Conn) {
	p.mux.Lock()
	delete(p.sm, c.Key())
	p.mux.Unlock()
}
