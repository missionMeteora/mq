package mq

import (
	"sync"

	"github.com/missionMeteora/jump/uuid"
)

func newReqWait() *reqWait {
	return &reqWait{
		m: make(map[uuid.UUID]ReqFunc),
	}
}

// reqWait is for holding onto a RespFunc while waiting for an outbound request
type reqWait struct {
	// TODO (Josh): See about utilizing the R functionality
	mux sync.RWMutex
	m   map[uuid.UUID]ReqFunc
}

// Get returns a RespFunc and an ok status
func (rw *reqWait) Get(id uuid.UUID) (fn ReqFunc, ok bool) {
	rw.mux.Lock()
	if fn, ok = rw.m[id]; ok {
		// If entry exists, we need to remove it from the list
		// Note: Think of the ReqFunc as being checked-in during Put
		// and checked-out during Get.
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
