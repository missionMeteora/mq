package pubsub

import (
	"sync"
	"time"

	"net"

	"github.com/missionMeteora/journaler"
	"github.com/missionMeteora/mq/conn"
	"github.com/missionMeteora/toolkit/errors"
)

// NewSub will accepts an address and a boolean connect on fail option and returns a new subscriber
func NewSub(addr string, cof bool) *Sub {
	var s Sub
	s.addr = addr
	s.cof = cof
	s.out = journaler.New("Sub")
	return &s
}

// Sub is a subscriber
type Sub struct {
	mux sync.RWMutex

	c   *conn.Conn
	out *journaler.Journaler

	// On connect functions
	onC []conn.OnConnectFn
	// On disconnect functions
	onDC []conn.OnDisconnectFn

	addr string
	cof  bool

	closed bool
}

func (s *Sub) reconnect() {
	var (
		c   *conn.Conn
		err error
	)

	for c == nil {
		if c, err = conn.NewClient(s.addr); err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
	}

	s.mux.Lock()
	if !s.closed {
		s.c = c
	}
	s.mux.Unlock()
}

// OnConnect will append an OnConnect func
func (s *Sub) OnConnect(fns ...conn.OnConnectFn) {
	s.mux.Lock()
	s.onC = append(s.onC, fns...)
	s.mux.Unlock()
}

// OnDisconnect will append an onDisconnect func
func (s *Sub) OnDisconnect(fns ...conn.OnDisconnectFn) {
	s.mux.Lock()
	s.onDC = append(s.onDC, fns...)
	s.mux.Unlock()
}

// Listen will listen for new messages
func (s *Sub) Listen(cb func([]byte) (end bool)) (err error) {
	var ended bool
	fn := func(b []byte) {
		if cb(b) {
			ended = true
		}
	}

	var nc net.Conn
	if nc, err = net.Dial("tcp", s.addr); err != nil {
		return
	}

	s.c = conn.New().OnConnect(s.onC...).OnDisconnect(s.onDC...)

	if err = s.c.Connect(nc); err != nil {
		return
	}

	for !ended {
		s.mux.RLock()
		if s.closed {
			err = errors.ErrIsClosed
		} else {
			err = s.c.Get(fn)
		}
		s.mux.RUnlock()

		if err != nil {
			s.out.Error("", err)
			break
		}
	}

	s.mux.RLock()
	if !s.closed && s.cof {
		go s.reconnect()
	}
	s.mux.RUnlock()
	return
}

// Close will close the subscriber
func (s *Sub) Close() (err error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if s.closed {
		return errors.ErrIsClosed
	}

	return s.c.Close()
}
