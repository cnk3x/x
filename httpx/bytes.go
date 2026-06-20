package httpx

import (
	"context"
	"errors"
	"io"
	"sync"
)

const (
	k32 = 1 << 15
	k64 = 1 << 16
	k1m = 1 << 20
)

type bytesPool struct {
	size int
	pool sync.Pool
}

var (
	b32k = newBytesPool(k32)
	b64k = newBytesPool(k64)
	b1m  = newBytesPool(k1m)
)

func newBytesPool(size int) *bytesPool {
	return &bytesPool{size: size, pool: sync.Pool{New: func() any {
		b := make([]byte, size)
		return &b
	}}}
}

func (p *bytesPool) Get() []byte {
	b := p.pool.Get().(*[]byte)
	return *b
}

func (p *bytesPool) Put(b []byte) {
	if cap(b) != p.size {
		return
	}
	clear(b)
	p.pool.Put(&b)
}

var (
	ErrInvalidWrite = errors.New("invalid write result")
	ErrShortWrite   = io.ErrShortWrite
)

func copyBuffer(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if buf == nil {
		size := 64 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}

		switch size {
		case k32:
			buf = b32k.Get()
			defer b32k.Put(buf)
		case k64:
			buf = b64k.Get()
			defer b64k.Put(buf)
		case k1m:
			buf = b1m.Get()
			defer b1m.Put(buf)
		default:
			buf = make([]byte, size)
		}
	}

	var written int64

	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		rn, re := src.Read(buf)

		if rn > 0 {
			wn, we := dst.Write(buf[0:rn])
			if wn < 0 || rn < wn {
				wn = 0
				if we == nil {
					we = ErrInvalidWrite
				}
			}

			written += int64(wn)

			if we != nil {
				return written, we
			}

			if rn != wn {
				return written, ErrShortWrite
			}
		}

		if re != nil {
			if re == io.EOF {
				re = nil
			}
			return written, re
		}
	}
}
