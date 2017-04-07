package pubsub

import (
	"github.com/missionMeteora/journaler"
	"github.com/missionMeteora/mq/conn"
)

// NewSub will return a new subscriber
func NewSub(addr string) (ss *Sub, err error) {
	var s Sub
	if s.c, err = conn.NewClient(addr); err != nil {
		return
	}

	ss = &s
	return
}

// Sub is a subscriber
type Sub struct {
	c   *conn.Conn
	out *journaler.Journaler
}

// Listen will listen for new messages
func (s *Sub) Listen(cb func([]byte) (end bool)) (ended bool) {
	fn := func(b []byte) {
		if cb(b) {
			ended = true
		}
	}

	for !ended {
		if err := s.c.Get(fn); err != nil {
			s.out.Error("", err)
			break
		}
	}

	return
}

// Close will close the subscriber
func (s *Sub) Close() (err error) {
	return s.c.Close()
}

/*

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
*/
