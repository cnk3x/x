package bus

import (
	"context"
	"iter"
	"sync"
	"sync/atomic"

	"github.com/cnk3x/x/ring"
)

type Bus[T BusEvent] struct {
	size     int
	clients  map[uint32]chan T
	cancels  map[uint32]context.CancelFunc
	history  ring.Ring[T]
	clientid atomic.Uint32
	evtid    atomic.Uint32 //小规模场景，够用。
	cancelWg sync.WaitGroup
	closed   atomic.Bool
	mu       sync.RWMutex
}

type BusEvent interface {
	SetID(id int)
	ID() int
}

func NewBus[T BusEvent](size, historySize int) *Bus[T] {
	b := &Bus[T]{
		size:    size,
		clients: make(map[uint32]chan T),
		cancels: make(map[uint32]context.CancelFunc),
	}
	if historySize > 0 {
		b.history.Reset(historySize)
	}
	return b
}

func (b *Bus[T]) Subscribe(ctx context.Context) (clientid uint32, events <-chan T) {
	if b.closed.Load() {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed.Load() {
		return 0, nil
	}

	clientid = b.clientid.Add(1)
	ch := make(chan T, b.size)
	b.clients[clientid] = ch

	ctx, cancel := context.WithCancel(ctx)
	b.cancels[clientid] = cancel

	b.cancelWg.Go(func() { <-ctx.Done(); b.internalUnsubscribe(clientid) })

	return clientid, ch
}

func (b *Bus[T]) Unsubscribe(clientid uint32) {
	if b.closed.Load() {
		return
	}
	b.internalUnsubscribe(clientid)
}

func (b *Bus[T]) internalUnsubscribe(clientid uint32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 清理 cancel（防止重复调用）
	if cancel, ok := b.cancels[clientid]; ok {
		cancel()
		delete(b.cancels, clientid)
	}

	// 清理 channel
	if ch, ok := b.clients[clientid]; ok {
		close(ch)
		delete(b.clients, clientid)
	}
}

func (b *Bus[T]) Publish(event T) int {
	if b.closed.Load() {
		return 0
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed.Load() {
		return 0
	}

	id := int(b.evtid.Add(1))
	event.SetID(id)
	if b.history.Cap() > 0 {
		b.history.Push(event)
	}

	for _, ch := range b.clients {
		select {
		case ch <- event:
		default:
		}
	}

	return id
}

func (b *Bus[T]) History(lastEventId int) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		if b.closed.Load() {
			return
		}

		b.mu.RLock()
		defer b.mu.RUnlock()

		var x int
		for _, event := range b.history.Items() {
			if event.ID() <= lastEventId {
				continue
			}
			if !yield(x, event) {
				return
			}
			x++
		}
	}
}

func (b *Bus[T]) Close() error {
	if !b.closed.CompareAndSwap(false, true) {
		return nil
	}

	b.mu.Lock()
	for _, cancel := range b.cancels {
		cancel()
	}

	if cap := b.history.Cap(); cap > 0 {
		b.history.Reset(cap)
	}
	b.mu.Unlock()

	b.cancelWg.Wait()
	return nil
}
