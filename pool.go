package mq

import (
	"sync"
	//	"github.com/missionMeteora/lockie"
)

const (
	psSmall  = 32
	psMedium = 64
	psLarge  = 128
	psXLarge = 256
)

func newPool() pool {
	return pool{
		{New: func() interface{} { return make([]byte, psSmall) }},
		{New: func() interface{} { return make([]byte, psMedium) }},
		{New: func() interface{} { return make([]byte, psLarge) }},
		{New: func() interface{} { return make([]byte, psXLarge) }},
	}
}

type pool [4]sync.Pool

func (p *pool) Get(sz int64) (out []byte) {
	switch {
	case sz <= psSmall:
		out = p[0].Get().([]byte)
	case sz <= psMedium:
		out = p[1].Get().([]byte)
	case sz <= psLarge:
		out = p[2].Get().([]byte)
	case sz <= psXLarge:
		out = p[3].Get().([]byte)
	default:
		out = make([]byte, 0, sz)
	}

	return
}

func (p *pool) Put(b []byte) {
	switch len(b) {
	case psSmall:
		p[0].Put(b)
	case psMedium:
		p[1].Put(b)
	case psLarge:
		p[2].Put(b)
	case psXLarge:
		p[3].Put(b)
	}
}
