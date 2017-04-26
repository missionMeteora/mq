package conn

import (
	//"bytes"
	"fmt"
	"io"
)

func newBuffer() *buffer {
	return &buffer{
		bs: make([]byte, 32*1024),
	}
}

type buffer struct {
	bs []byte
	n  int
}

// ReadN will read n bytes from an io.Reader
// Note: Internal byteslice resets on each read
func (b *buffer) ReadN(r io.Reader, n uint64) (err error) {
	if int(n) > len(b.bs) {
		fmt.Println("GROW", n, len(b.bs))
		// Our internal slice is too small, grow before reading
		b.bs = make([]byte, n)
	}

	b.n, err = io.ReadFull(r, b.bs[:n])
	return
}

func (b *buffer) Bytes() []byte {
	return b.bs[:b.n]
}
