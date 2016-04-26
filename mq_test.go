package mq

import (

	//"fmt"
	//"os"
	"testing"
	"time"
)

var (
	s  *Server
	c  *Client
	ti testItem

	val     []byte
	respVal []byte

	stmnt = []byte("Hai there!")
	req   = []byte("How are you?")

	clntName     = "MoodyMoose"
	clntChunk    = NewChunkFromString(clntName)
	clntTkn      = "DingDong"
	clntTknChunk = NewChunkFromString(clntTkn)

	srvName  = "HonestHyena"
	srvChunk = NewChunkFromString(srvName)
)

type testItem struct{}

func (t *testItem) Statement(b []byte) {
	val = b
}

func (t *testItem) Response(b []byte) (r []byte) {
	val = b
	return []byte{'o', 'k'}
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

	if c, err = NewClient(":1337", clntChunk, clntTknChunk, nil); err != nil {
		t.Error("Error getting new client", err)
		return
	}

	if s, err = NewServer(":1337", srvChunk, op, KeyToken{clntName, clntTkn}); err != nil {
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

	if s, err = NewServer(":1337", srvChunk, op, KeyToken{clntName, clntTkn}); err != nil {
		t.Error("Error getting new server", err)
		return
	}

	if c, err = NewClient(":1337", clntChunk, clntTknChunk, nil); err != nil {
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

	if c, err = NewClient(":1337", clntChunk, clntTknChunk, nil); err != nil {
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

/*
func TestMain(m *testing.M) {
	var err error

	connected := make(chan struct{}, 1)
	op := NewOp(
		func(ch Chunk) error {
			if ch == clntChunk {
				connected <- struct{}{}
			}

			return nil
		},
		func(ch Chunk) {
			fmt.Println(ch, "disconnected")
		},
	)

	if s, err = NewServer(":1337", srvChunk, op); err != nil {
		fmt.Println("Error getting new server", err)
	}

	s.PutAuth(clntName, clntTkn)

	if c, err = NewClient(":1337", clntChunk, clntTknChunk, nil); err != nil {
		fmt.Println("Error getting new client", err)
	}

	<-connected

	s.Close()

	if s, err = NewServer(":1337", srvChunk, op); err != nil {
		fmt.Println("Error getting new server", err)
	}

	s.PutAuth(clntName, clntTkn)

	fmt.Println("Waiting for new connection")
	<-connected
	fmt.Println("Connected!")

	go func() {
		for c.Receive(&ti) == nil {
		}
	}()

	os.Exit(m.Run())
}
*/

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
