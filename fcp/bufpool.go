package fcp

import (
	"github.com/cnk3x/x/pool"
)

const (
	s32K = 320 * 1024
	s1M  = 1024 * 1024
	s4M  = 4 * 1024 * 1024
)

var (
	B32K = createPool(s32K)
	B1M  = createPool(s1M)
	B4M  = createPool(s4M)
)

func createPool(size int) *pool.Pool[[]byte] {
	return pool.NewPool(
		func() []byte { return make([]byte, size) },
		func(b []byte) ([]byte, bool) {
			if cap(b) == size {
				return b[:size], true
			}
			return nil, false
		},
	)
}
