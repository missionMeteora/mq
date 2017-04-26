package conn

import "io"

type buffer struct {
	bs  []byte
	len uint64
	n   int
}

// ReadN will read n bytes from an io.Reader
// Note: Internal byteslice resets on each read
func (b *buffer) ReadN(r io.Reader, n uint64) (err error) {
	var rn int
	ttl := int(n)
	b.n = 0
	if n > b.len {
		// Our internal slice is too small, grow before reading
		b.bs = make([]byte, n)
		b.len = n
	}

	for b.n < ttl && err == nil {
		rn, err = r.Read(b.bs[b.n:b.len])
		b.n += rn
	}

	if err == io.EOF && b.n == ttl {
		err = nil
	}

	return
}

func (b *buffer) Bytes() []byte {
	return b.bs[:b.n]
}
