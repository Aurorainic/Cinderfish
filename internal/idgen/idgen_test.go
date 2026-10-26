package idgen

import (
	"strings"
	"testing"
	"time"
)

func TestLengthAndCharset(t *testing.T) {
	for range 100 {
		id := New()
		if len(id) != 18 {
			t.Fatalf("id length = %d, want 18", len(id))
		}
		for _, r := range id {
			if !strings.ContainsRune(base36, r) {
				t.Fatalf("id %q contains non-base36 char %q", id, r)
			}
		}
	}
}

func TestTimeSortable(t *testing.T) {
	t1 := time.UnixMilli(1700000000000)
	t2 := time.UnixMilli(1700000001000)

	a, b := NewAt(t1), NewAt(t2)
	if a >= b {
		t.Fatalf("ids not time-ordered: %s >= %s", a, b)
	}

	// Same millisecond: order is random but prefix must match.
	c, d := NewAt(t1), NewAt(t1)
	if c[:10] != d[:10] {
		t.Fatalf("same-millisecond ids share timestamp prefix: %s vs %s", c, d)
	}
}

func TestRandomnessUnique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for range 1000 {
		id := New()
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
}
