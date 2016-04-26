package mq

import (
	"sync"

	"github.com/missionMeteora/lockie"
)

const (
	// Number of bytes in a kilobyte
	kb = 1024
)

func newPool() *pool {
	p := pool{
		mux: lockie.NewLockie(),

		// 32 bytes is our smallest buffer size
		head: 32,
		// Make internal store
		m: make(map[int64]*sync.Pool, 12),
	}

	// Setup cache key for 32 byte buffers
	p.appendKey(32)
	// Setup cache key for 64 byte buffers with a len of 128 and a cap of 8192
	p.appendKey(64)
	return &p
}

type pool struct {
	mux lockie.Lockie

	// Smallest buffer size
	head int64
	// Largest buffer size
	tail int64
	// List of keys
	keys []int64

	// Internal store of pools by buffer size
	m map[int64]*sync.Pool
}

// appendKey appends a pool to the pool storage by the provided key
func (p *pool) appendKey(k int64) {
	// Set tail to new key
	p.tail = k
	// Append key to keys list
	p.keys = append(p.keys, p.tail)

	// Create pool as value for the provided key within the internal storage
	p.m[k] = &sync.Pool{
		New: func() interface{} {
			return make([]byte, k)
		},
	}
}

func (p *pool) getNext(sz int64) int64 {
	// For tail multiplied by our multiplier is less than the requested size
	for p.tail < sz {
		// Multiply our multiplier by four
		p.tail *= 4
		p.appendKey(p.tail)
	}

	// Return current multiplied by our multiplier
	return p.tail
}

func (p *pool) Get(sz int64) (b []byte) {
	p.mux.Lock()
	// Iterate through keys
	for _, k := range p.keys {
		if sz > k {
			// Size is greater than our key, continue
			continue
		}

		if bp, ok := p.m[k]; ok {
			b = bp.Get().([]byte)
			// We have our value, break
			break
		} else {
			// This shouldn't be happening, panic
			panic("Value does not exist at requested key")
		}
	}

	if sz > p.tail {
		p.getNext(sz)
		b = p.m[p.tail].Get().([]byte)
	}

	p.mux.Unlock()
	return

}

func (p *pool) Put(b []byte) {
	p.mux.Lock()
	// Get pool channel for this slice length
	if bp, ok := p.m[int64(len(b))]; ok {
		// Pool channel exists, return byteslice to the pool
		bp.Put(b)
	}
	p.mux.Unlock()
	return
}
