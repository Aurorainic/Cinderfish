// Package idgen produces Misskey-style string ids: a base36 millisecond
// timestamp (10 chars) followed by 8 base36 random chars. Ids sort
// chronologically as strings, matching timeline cursor semantics of
// sinceId/untilId.
package idgen

import (
	"crypto/rand"
	"time"
)

const base36 = "0123456789abcdefghijklmnopqrstuvwxyz"

// New returns a new id for the current time.
func New() string {
	return NewAt(time.Now())
}

// NewAt returns a new id stamped with t's millisecond timestamp.
func NewAt(t time.Time) string {
	ms := uint64(t.UnixMilli()) & ((1 << 48) - 1)

	out := make([]byte, 18)
	// 48 bits fit in 10 base36 digits (36^10 > 2^51); leading digit
	// stays '0' until the year 2158, which keeps ids time-sortable.
	for i := 9; i >= 0; i-- {
		out[i] = base36[ms%36]
		ms /= 36
	}

	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("idgen: crypto/rand failed: " + err.Error())
	}
	for i := 0; i < 8; i++ {
		out[10+i] = base36[int(b[i])%36]
	}
	return string(out)
}
