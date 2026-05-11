package pool

import "sync"

type Pool[T any] struct {
	s      sync.Pool
	create func() T
	reset  func(T) (T, bool)
}

func NewPool[T any](create func() T, reset func(T) (T, bool)) *Pool[T] {
	return (&Pool[T]{}).Init(create, reset)
}

func (p *Pool[T]) Init(create func() T, reset func(T) (T, bool)) *Pool[T] {
	p.create = create
	p.reset = reset
	p.s.New = func() any { return p.create() }
	return p
}

func (p *Pool[T]) Get() (value T, put func()) {
	value = p.s.Get().(T)
	put = func() { p.put(value) }
	return
}

func (p *Pool[T]) put(v T) {
	if p.reset == nil {
		p.s.Put(v)
		return
	}
	if v, ok := p.reset(v); ok {
		p.s.Put(v)
	}
}
