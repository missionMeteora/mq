package main

import (
	"fmt"
	"time"

	"github.com/missionMeteora/jump/mq"
	"github.com/missionMeteora/misc/pubsub"
	"github.com/pkg/profile"
)

var (
	val []byte

	stmnt = []byte("Hai there!")
	req   = []byte("How are you?")

	clntName     = "MoodyMoose"
	clntChunk    = mq.NewChunkFromString(clntName)
	clntTkn      = "DingDong"
	clntTknChunk = mq.NewChunkFromString(clntTkn)

	srvName  = "HonestHyena"
	srvChunk = mq.NewChunkFromString(srvName)

	iter = 2000000

	doCompare = false
	doProfile = false
)

type testItem struct{}

var cnt int
var puupuu = make(chan struct{}, 1)

func (t *testItem) Statement(b []byte) {
	cnt++
	if cnt%(iter/10) == 0 {
		fmt.Println("Statement!", cnt)
	}
	val = b
}

func (t *testItem) Response(b []byte) (r []byte) {
	val = b
	return []byte{'o', 'k'}
}

func (t *testItem) Sub(b []byte, err error) error {
	cnt++
	if cnt%(iter/10) == 0 {
		fmt.Println("Sub!", cnt)
	}

	val = b

	if cnt == iter {
		puupuu <- struct{}{}
	}
	return nil
}

func main() {
	run("cpu")
	//run("mem")
	//run("block")
}

func run(t string) {
	var (
		s  *mq.Server
		c  *mq.Client
		ti testItem

		st, et int64

		pu *pubsub.Pub

		err error
		p   interface {
			Stop()
		}
	)

	if doProfile {
		switch t {
		case "cpu":
			p = profile.Start(profile.CPUProfile, profile.ProfilePath("./output/"), profile.NoShutdownHook)
		case "mem":
			p = profile.Start(profile.MemProfile, profile.ProfilePath("./output/"), profile.NoShutdownHook)
		case "block":
			p = profile.Start(profile.BlockProfile, profile.ProfilePath("./output/"), profile.NoShutdownHook)
		}
	}

	connected := make(chan struct{}, 1)
	op := mq.NewOp(
		func(ch mq.Chunk) error {
			if ch == clntChunk {
				connected <- struct{}{}
			}

			return nil
		},
		func(ch mq.Chunk) {
			fmt.Println(ch, "disconnected")
		},
	)

	if s, err = mq.NewServer(":1337", srvChunk, op); err != nil {
		fmt.Println("Error getting new server", err)

		goto CLOSE
	}

	s.PutAuth(clntName, "DingDong")
	fmt.Println("Server started")

	if c, err = mq.NewClient(":1337", clntChunk, clntTknChunk, op); err != nil {
		fmt.Println("Error getting new client", err)

		goto CLOSE
	}

	fmt.Println("Waiting on client")
	<-connected

	fmt.Println("Connection established, mq test starting")
	st = time.Now().UnixNano()
	go func() {
		for i := 0; i < iter; i++ {
			if err := s.Statement(clntName, req); err != nil {
				fmt.Println("Error with statement!", err)
			}
		}

		s.Close()
	}()

	for err == nil {
		err = c.Receive(&ti)
		if err != nil {
			fmt.Println("Rec err", err)
		}
	}
	et = time.Now().UnixNano()
	c.Close()
	fmt.Println("Total time:", (et-st)/1000000, "ms")

	if !doCompare {
		goto CLOSE
	}

	cnt = 0

	if pu, err = pubsub.NewPublisher("tcp://localhost:1338"); err != nil {
		fmt.Println("Error getting new publisher", err)

		goto CLOSE
	}

	if _, err = pubsub.NewSubscriber("tcp://localhost:1338", ti.Sub); err != nil {
		fmt.Println("Error getting new subscriber", err)

		goto CLOSE
	}

	fmt.Println("Waiting for connection to establish")
	<-time.NewTimer(time.Millisecond * 1000).C

	fmt.Println("Connection established, mq test starting")
	st = time.Now().UnixNano()

	for i := 0; i < iter; i++ {
		if err = pu.Publish(req); err != nil {
			fmt.Println("Error publishing!", err)
		}
	}

	<-puupuu

	if err = pu.Close(); err != nil {
		fmt.Println("Error closing publisher", p)
	}

	et = time.Now().UnixNano()
	fmt.Println("Mangos Total time:", (et-st)/1000000, "ms")

CLOSE:
	if doProfile {
		p.Stop()
	}
}
