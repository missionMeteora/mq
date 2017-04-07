package pubsub

import (
	"sync"
	"testing"
	"time"
)

var testVal = []byte("hello world!")

func TestPubSub(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		var (
			p   *Pub
			err error
		)

		defer wg.Done()

		if p, err = NewPub(":16777"); err != nil {
			t.Fatal(err)
		}

		time.Sleep(time.Millisecond * 30)
		p.Put(testVal)
		p.Put(testVal)
		p.Put(testVal)
		p.Close()
	}()

	time.Sleep(time.Millisecond * 10)

	go func() {
		var (
			s   *Sub
			cnt int
			err error
		)

		defer wg.Done()

		if s, err = NewSub(":16777"); err != nil {
			t.Fatal(err)
		}

		s.Listen(func(b []byte) bool {
			var msg = string(b)
			if msg != string(testVal) {
				t.Fatalf("invalid message, expected '%s' and received '%s'", "hello world!", msg)
			}

			if cnt++; cnt == 3 {
				return true
			}

			return false
		})

		if cnt != 3 {
			t.Fatalf("invalid count, expected %v and received %v", 3, cnt)
		}
	}()

	wg.Wait()
}
