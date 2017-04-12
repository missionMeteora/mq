package conn

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

var (
	testVal    = []byte("hello world")
	testSetVal []byte
)

func TestConn(t *testing.T) {
	done := make(chan error, 1)

	go func() {
		var (
			s   *Conn
			l   net.Listener
			nc  net.Conn
			msg string
			err error
		)

		if l, err = net.Listen("tcp", ":1337"); err != nil {
			return
		}

		if nc, err = l.Accept(); err != nil {
			return
		}

		s = New()
		if err = s.Connect(nc); err != nil {
			return
		}

		if err = s.Put([]byte("hello 0")); err != nil {
			done <- err
		}

		if msg, err = s.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 1" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		if err = s.Put([]byte("hello 2")); err != nil {
			done <- err
		}

		if msg, err = s.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 3" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		done <- nil
	}()

	time.Sleep(time.Millisecond * 10)

	go func() {
		var (
			c   *Conn
			nc  net.Conn
			msg string
			err error
		)

		if nc, err = net.Dial("tcp", ":1337"); err != nil {
			done <- err
		}

		c = New()
		c.Connect(nc)

		if msg, err = c.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 0" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		if err = c.Put([]byte("hello 1")); err != nil {
			done <- err
		}

		if msg, err = c.GetStr(); err != nil {
			done <- err
		} else if msg != "hello 2" {
			done <- fmt.Errorf("invalid message, expected '%s' recevied '%s'", "hello 0", msg)
		}

		if err = c.Put([]byte("hello 3")); err != nil {
			done <- err
		}

		// Do close shit here
	}()

	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func BenchmarkMQ(b *testing.B) {
	var (
		s = New()
		c = New()

		ready = make(chan struct{}, 1)

		l   net.Listener
		wg  sync.WaitGroup
		err error
	)

	wg.Add(b.N)

	go func() {
		var (
			nc  net.Conn
			err error
		)

		if l, err = net.Listen("tcp", ":1337"); err != nil {
			b.Fatal(err)
		}

		ready <- struct{}{}

		if nc, err = l.Accept(); err != nil {
			b.Fatal(err)
		}

		if err = s.Connect(nc); err != nil {
			b.Fatal(err)
		}

		ready <- struct{}{}
	}()

	<-ready

	go func() {
		var (
			nc  net.Conn
			err error
		)

		if nc, err = net.Dial("tcp", ":1337"); err != nil {
			b.Fatal(err)
		}

		if err = c.Connect(nc); err != nil {
			b.Fatal(err)
		}

		ready <- struct{}{}

		var n int
		for {
			if c.Get(func(b []byte) {
				testSetVal = b
				n++
				wg.Done()
			}) != nil {
				return
			}
		}
	}()

	<-ready
	<-ready
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err = s.Put(testVal); err != nil {
			b.Fatal(err)
		}
	}

	wg.Wait()
	b.ReportAllocs()
	s.Close()
	c.Close()
	l.Close()
}
