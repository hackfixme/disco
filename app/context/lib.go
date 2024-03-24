package context

import (
	"github.com/nrednav/cuid2"
)

// NewUUIDGenerator returns a function that generates UUIDs of the given length.
func NewUUIDGenerator(length int) (func() string, error) {
	return cuid2.Init(cuid2.WithLength(length))
}
