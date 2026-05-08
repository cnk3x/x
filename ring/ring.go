package ring

import (
	"iter"
)

// Ring 是一个固定容量的环形缓冲区（循环队列）
// 当缓冲区满时，新元素会覆盖最旧的元素
// 该实现不是并发安全的，如需并发使用请添加互斥锁
type Ring[T any] struct {
	count int // 实际元素数量
	index int // 下一个写入位置
	data  []T // 底层数组，长度 = 容量
}

// NewRing 创建一个指定容量的环形缓冲区
func NewRing[T any](size int) *Ring[T] {
	var r Ring[T]
	r.Reset(size)
	return &r
}

// Reset 重置缓冲区并设置新容量，丢弃现有数据
func (r *Ring[T]) Reset(size int) {
	if size <= 0 {
		panic("size must be positive")
	}
	r.count, r.index = 0, 0
	if cap := len(r.data); size > cap {
		r.data = append(r.data, make([]T, size-cap)...)
	} else {
		r.data = r.data[:size]
	}
}

// Push 添加一个元素到缓冲区
// 如果缓冲区已满，会覆盖最旧的元素
func (r *Ring[T]) Push(item T) {
	size := len(r.data)
	if size == 0 {
		return
	}
	r.count = min(size, r.count+1)
	r.data[r.index] = item
	r.index = (r.index + 1) % size
}

// Items 返回一个迭代器，按从旧到新的顺序遍历所有元素
// 每个元素会返回其索引（0-based）和值
func (r *Ring[T]) Items() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		if r.count == 0 {
			return
		}
		cap := len(r.data)
		start := (r.index + cap - r.count) % cap
		for i := 0; i < r.count; i++ {
			if !yield(i, r.data[(start+i)%cap]) {
				return
			}
		}
	}
}

// Len 返回当前元素数量
func (r *Ring[T]) Len() int { return r.count }

// Cap 返回容量
func (r *Ring[T]) Cap() int { return len(r.data) }
