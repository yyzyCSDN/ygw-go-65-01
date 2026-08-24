package model

import (
	"strings"
	"testing"
	"time"
)

func TestNewCallCopiesPayload(t *testing.T) {
	payload := []byte("hello")
	call := NewCall("c1", "echo", payload, time.Now().Add(time.Second))
	payload[0] = 'H'
	if !strings.HasPrefix(string(call.Payload), "hello") {
		t.Fatalf("payload must be copied, got %q", call.Payload)
	}
	if call.Status != StatusQueued || call.Attempt != 1 {
		t.Fatalf("unexpected initial state: %s/%d", call.Status, call.Attempt)
	}
}

func TestHashIDStableAndDistinct(t *testing.T) {
	a := HashID("fn", "k1")
	b := HashID("fn", "k1")
	c := HashID("fn", "k2")
	if a != b || a == c {
		t.Fatalf("hash must be stable and distinct: %s %s %s", a, b, c)
	}
}
