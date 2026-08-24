package model

import (
	"fmt"
	"io"

	"github.com/cespare/xxhash/v2"
)

// HashID builds a stable identifier from the given parts using xxhash.
func HashID(parts ...string) string {
	h := xxhash.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
		_, _ = io.WriteString(h, "\x00")
	}
	return fmt.Sprintf("%016x", h.Sum64())
}
