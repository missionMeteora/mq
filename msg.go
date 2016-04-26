package mq

import (
	"unsafe"

	"github.com/missionMeteora/binny.v2"
	"github.com/missionMeteora/jump/uuid"
	"github.com/missionMeteora/mq/internal/common"
)

type msg struct {
	id uuid.UUID
	t  common.MsgType
	s  common.Status

	body []byte
}

// Bytes will return a representation of it's contents in the form of a byteslice
func (m *msg) Bytes(b []byte) (out []byte, n int) {
	blen := int64(len(m.body))
	// Set the message type at index 24
	b[24] = byte(m.t)
	b[25] = byte(m.s)

	// Copy id from index zero to (not including) index sixteen
	copy(b[:16], m.id[:])
	// Copy body length value (as a byteslice) from index sixteen to index twenty-four (not including)
	copy(b[16:24], (*[8]byte)(unsafe.Pointer(&blen))[:])
	if m.body != nil {
		// If body exists for message, copy body from index twenty-five until the end of the body
		copy(b[26:], m.body)
	}

	// Return populated byteslice
	return b, int(common.HeaderLen + blen)
}

func (m *msg) MarshalBinny(enc *binny.Encoder) (err error) {
	if err = enc.Encode(&m.id); err != nil {
		return
	}

	if err = enc.WriteUint8(uint8(m.t)); err != nil {
		return
	}

	if err = enc.WriteBytes(m.body[:]); err != nil {
		return
	}

	return
}

func (m *msg) UnmarshalBinny(dec *binny.Decoder) (err error) {
	if err = dec.Decode(&m.id); err != nil {
		return
	}

	var t uint8
	if t, err = dec.ReadUint8(); err != nil {
		return
	}

	m.t = common.MsgType(t)

	if m.body, err = dec.ReadBytes(); err != nil {
		return
	}

	return
}
