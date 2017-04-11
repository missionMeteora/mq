package conn

import (
	"fmt"
	"testing"
	"time"
)

func TestConn(t *testing.T) {
	done := make(chan error, 1)

	go func() {
		var (
			s   *Conn
			msg string
			err error
		)

		if s, err = NewServer(":1337"); err != nil {
			done <- err
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
			nc net.Conn
			msg string
			err error
		)

		var 
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
