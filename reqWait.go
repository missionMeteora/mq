package mq

import (
	"github.com/missionMeteora/jump/uuid"
	"github.com/missionMeteora/lockie"
)

func newReqWait() *reqWait {
	return &reqWait{
		mux: lockie.NewLockie(),
		m:   make(map[uuid.UUID]ReqFunc),
	}
}

// reqWait is for holding onto a RespFunc while waiting for an outbound request
type reqWait struct {
	mux lockie.Lockie
	m   map[uuid.UUID]ReqFunc
}

// Get returns a RespFunc and an ok status
func (rw *reqWait) Get(id uuid.UUID) (fn ReqFunc, ok bool) {
	rw.mux.Lock()
	if fn, ok = rw.m[id]; ok {
		// If entry exists, delete it
		delete(rw.m, id)
	}
	rw.mux.Unlock()
	return
}

// Put will set key of id with a value of the argument-provided fn
func (rw *reqWait) Put(id uuid.UUID, fn ReqFunc) {
	rw.mux.Lock()
	rw.m[id] = fn
	rw.mux.Unlock()
}

// Dump clear our current reqWait list. Intended to be used on close by the parent
func (rw *reqWait) Dump() {
	rw.mux.Lock()
	for _, fn := range rw.m {
		// Dumping all waiting functions with nil
		fn(nil)
	}

	// Replace map completely
	rw.m = make(map[uuid.UUID]ReqFunc)
	rw.mux.Unlock()
}
