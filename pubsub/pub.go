package pubsub

import (
	"net"
	"sync"
	"time"

	"github.com/missionMeteora/journaler"
	"github.com/missionMeteora/mq/conn"
	"github.com/missionMeteora/toolkit/errors"
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

	// On connect functions
	onC []conn.OnConnectFn
	// On disconnect functions
	onDC []conn.OnDisconnectFn

	closed bool
}

// Listen will listen for inbound subscribers
func (p *Pub) listen(l net.Listener) {
	var err error

	for {
		var nc net.Conn
		if nc, err = l.Accept(); err != nil {
			p.out.Error("", err)
			continue
		}

		c := conn.NewConn()
		c.OnDisconnect(p.remove)
		c.Connect(nc)

		p.mux.Lock()
		p.sm[c.Key()] = c
		p.mux.Unlock()
	}
}

// OnConnect will append an OnConnect func
func (p *Pub) OnConnect(fn conn.OnConnectFn) {
	p.mux.Lock()
	p.onC = append(p.onC, fn)
	p.mux.Unlock()
}

// OnDisconnect will append an onDisconnect func
func (p *Pub) OnDisconnect(fn conn.OnDisconnectFn) {
	p.mux.Lock()
	p.onDC = append(p.onDC, fn)
	p.mux.Unlock()
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

func (p *Pub) getConn(key string) (c *conn.Conn, ok bool) {
	p.mux.RLock()
	c, ok = p.get(key)
	p.mux.RUnlock()
	return

}

// Remove will remove a subscriber
func (p *Pub) Remove(key string) (err error) {
	c, ok := p.getConn(key)
	if !ok {
		return
	}

	if err = c.Close(); err != nil {
		// We had an error closing the connection. Let's ensure we properly remove the connection from our list
		p.remove(c)
	}

	return
}

func (p *Pub) close(wg *sync.WaitGroup) (errs *errors.ErrorList) {
	errs = &errors.ErrorList{}
	if p.closed {
		errs.Push(errors.ErrIsClosed)
		return
	}

	wg.Add(len(p.sm))
	for _, s := range p.sm {
		go func(c *conn.Conn) {
			errs.Push(c.Close())
			wg.Done()
		}(s)
	}

	return
}

// Close will close the Pubber
func (p *Pub) Close() error {
	var wg sync.WaitGroup
	p.mux.Lock()
	errs := p.close(&wg)
	p.mux.Unlock()
	wg.Wait()
	return errs.Err()
}

func (p *Pub) remove(c *conn.Conn) {
	p.mux.Lock()
	delete(p.sm, c.Key())
	p.mux.Unlock()
}
