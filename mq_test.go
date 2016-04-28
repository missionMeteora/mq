package mq

import (
	"fmt"
	"os"
	"testing"
	"time"
)

const (
	clntName = "MoodyMoose"
	clntTkn  = "DingDong"

	mainPort = ":1337"
	altPort  = ":1338"
	srvName  = "HonestHyena"
)

var (
	s  *Server
	c  *Client
	ti testItem

	val     []byte
	respVal []byte

	stmnt = []byte("Hai there!")
	req   = []byte("How are you?")

	clntChunk, _    = NewChunkFromString(clntName)
	clntTknChunk, _ = NewChunkFromString(clntTkn)
	srvChunk, _     = NewChunkFromString(srvName)
)

type testItem struct{}

func (t *testItem) Statement(b []byte) {
	val = b
}

func (t *testItem) Response(b []byte) (r []byte) {
	val = b
	return []byte{'o', 'k'}
}

func TestMain(m *testing.M) {
	var err error
	connected := make(chan struct{}, 1)
	op := NewOp(
		func(ch Chunk) error {
			fmt.Println("Hai!", ch.String())
			if ch == clntChunk {
				connected <- struct{}{}
			}

			return nil
		},
		func(ch Chunk) {
			fmt.Println(ch, "disconnected")
		},
	)

	if s, err = NewServer(mainPort, srvChunk, op); err != nil {
		fmt.Println("Error getting new server", err)
		return
	}

	s.PutAuth(clntName, clntTkn)

	if c, err = NewClient(mainPort, clntChunk, clntTknChunk, nil); err != nil {
		fmt.Println("Error getting new client", err)
		return
	}

	<-connected

	go func() {
		for c.Receive(&ti) == nil {
		}
	}()

	os.Exit(m.Run())
}

func TestClientFirst(t *testing.T) {
	var (
		s   *Server
		c   *Client
		err error
	)

	connected := make(chan struct{}, 1)
	op := NewOp(func(ch Chunk) error {
		connected <- struct{}{}
		return nil
	}, nil)

	if c, err = NewClient(altPort, clntChunk, clntTknChunk, nil); err != nil {
		t.Error("Error getting new client", err)
		return
	}

	if s, err = NewServer(altPort, srvChunk, op, KeyToken{clntName, clntTkn}); err != nil {
		t.Error("Error getting new server", err)
		return
	}

	<-connected

	c.Close()
	s.Close()
	time.Sleep(time.Second * 1)
}

func TestClientReconnect(t *testing.T) {
	var (
		s   *Server
		c   *Client
		err error

		msg  = "Hai!"
		msgB = []byte(msg)
	)

	connected := make(chan struct{}, 1)
	op := NewOp(func(ch Chunk) error {
		connected <- struct{}{}
		return nil
	}, nil)

	if s, err = NewServer(altPort, srvChunk, op, KeyToken{clntName, clntTkn}); err != nil {
		t.Error("Error getting new server", err)
		return
	}

	if c, err = NewClient(altPort, clntChunk, clntTknChunk, nil); err != nil {
		t.Error("Error getting new client", err)
		return
	}

	<-connected
	s.Statement(clntChunk.String(), msgB)
	c.Receive(NewRec(nil, func(b []byte) {
		if str := string(b); str != msg {
			t.Errorf("Incorrect message, expected \"%s\" and we received \"%s\"", msg, str)
		}
	}))
	c.Close()

	if c, err = NewClient(altPort, clntChunk, clntTknChunk, nil); err != nil {
		t.Error("Error getting new client", err)
		return
	}

	<-connected
	s.Statement(clntChunk.String(), msgB)
	c.Receive(NewRec(nil, func(b []byte) {
		if str := string(b); str != msg {
			t.Errorf("Incorrect message, expected \"%s\" and we received \"%s\"", msg, str)
		}
	}))

	c.Close()
	s.Close()
	time.Sleep(time.Second * 1)
}

func BenchmarkStatement(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s.Statement(clntName, stmnt)
	}

	b.ReportAllocs()
}

func BenchmarkRequest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s.Request(clntName, req, func(b []byte) {
			respVal = b
		})
	}

	b.ReportAllocs()
}
