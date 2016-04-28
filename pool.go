package mq

import (
	"sync"
	//	"github.com/missionMeteora/lockie"
)

func newPool() pool {
	return pool{
		{New: func() interface{} { return make([]byte, 64) }},
		{New: func() interface{} { return make([]byte, 128) }},
		{New: func() interface{} { return make([]byte, 256) }},
		{New: func() interface{} { return make([]byte, 512) }},
	}
}

type pool [4]sync.Pool

func (p *pool) Get(sz int64) (out []byte) {
	switch {
	case sz < 65:
		out = p[0].Get().([]byte)
	case sz < 129:
		out = p[1].Get().([]byte)
	case sz < 257:
		out = p[2].Get().([]byte)
	case sz < 513:
		out = p[3].Get().([]byte)
	default:
		out = make([]byte, 0, sz)
	}

	return
}

func (p *pool) Put(b []byte) {
	switch len(b) {
	case 64:
		p[0].Put(b)
	case 128:
		p[1].Put(b)
	case 256:
		p[2].Put(b)
	case 512:
		p[3].Put(b)
	}
}
