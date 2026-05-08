// Package httpx 提供增强的 HTTP 功能，包括进度监控等。
package httpx

import (
	"io"
	"net/http"
	"sync"
	"time"
)

// ProgressWriter 实现了一个带有滑动窗口速度计算功能的 io.Writer 装饰器。
// 它能够实时监控数据传输进度、速度（BPS）以及预计剩余时间（ETA）。
type ProgressWriter struct {
	parent   io.Writer
	data     []int64                   // window data: 存储采样点的瞬时速度
	count    int                       // window count: 已填充的采样点数量
	cur      int                       // window index: 环形缓冲区的当前指针
	lastTime time.Time                 // window time: 上一次计算速度的时间点
	delta    int64                     // delta after lastTime: 采样周期内累积的数据量
	state    ProgressState             // meter state: 当前的进度快照
	report   func(state ProgressState) // 进度回调函数
	mu       sync.Mutex                // 保护 state 和采样数据的互斥锁
}

// ProgressState 描述了某一时刻数据传输的完整状态。
type ProgressState struct {
	Total      int64         // Total 表示总字节数（Content-Length）
	Downloaded int64         // Downloaded 表示已下载/写入的字节数
	Speed      int64         // Speed 表示平均下载速度（比特每秒，bps）
	Eta        time.Duration // Eta 表示预计剩余时间
}

// NewProgress 创建并返回一个新的进度监控器。
// parent 是实际执行写入操作的 io.Writer；
// total 是待传输的总字节数，如果未知可传入 -1；
// report 是一个回调函数，每当状态更新时会被调用。
func NewProgress(parent io.Writer, total int64, report func(state ProgressState)) *ProgressWriter {
	return &ProgressWriter{
		parent: parent,
		data:   make([]int64, 30),
		report: report,
		state:  ProgressState{Total: total},
	}
}

// SaveResponse 是一个辅助函数，用于将 http.Response 的 Body 流式写入指定的 Writer，
// 同时通过报告函数反馈传输进度。它内部使用了 512KB 的缓冲区以优化性能。
func SaveResponse(w io.Writer, resp *http.Response, report func(state ProgressState)) error {
	return iocopy(NewProgress(w, resp.ContentLength, report), resp.Body)
}

// Write 实现了 io.Writer 接口。
// 它会将数据写入底层的 parent writer，并根据采样周期（500ms）更新传输速度和 ETA。
func (meter *ProgressWriter) Write(p []byte) (n int, err error) {
	if meter.parent != nil {
		if n, err = meter.parent.Write(p); err != nil {
			return
		}
	} else {
		n = len(p)
	}

	delta := int64(n)
	now := time.Now()

	meter.mu.Lock()
	meter.state.Downloaded += delta
	meter.delta += delta

	if meter.lastTime.IsZero() {
		meter.lastTime = now
		// 初始化状态
		meter.state.Speed = -1
		meter.state.Eta = -1
		st := meter.state
		meter.mu.Unlock()
		if meter.report != nil {
			meter.report(st)
		}
		return
	}

	interval := now.Sub(meter.lastTime) // 提前定义 interval

	// 检查是否达到采样周期 (500ms)，除非下载已完成
	isFinished := meter.state.Total > 0 && meter.state.Downloaded >= meter.state.Total
	if !isFinished && interval < 500*time.Millisecond {
		st := meter.state
		meter.mu.Unlock()
		if meter.report != nil {
			meter.report(st)
		}
		return
	}

	// --- 开始计算速度 ---
	// 防御性检查：确保 interval 不为 0 避免 NaN/Inf
	seconds := interval.Seconds()
	if seconds > 0 {
		meter.cur = (meter.cur + 1) % len(meter.data)
		meter.count = min(len(meter.data), meter.count+1)
		meter.data[meter.cur] = int64(float64(meter.delta) / seconds)

		var speedSum int64
		for _, v := range meter.data[:meter.count] {
			speedSum += v
		}
		meter.state.Speed = speedSum / int64(meter.count)
	}

	// 计算 ETA
	if meter.state.Total > 0 && meter.state.Speed > 0 {
		rem := max(meter.state.Total-meter.state.Downloaded, 0)
		meter.state.Eta = time.Duration(rem * int64(time.Second) / meter.state.Speed)
	} else {
		meter.state.Eta = -1
	}

	// 重置采样窗口
	meter.lastTime = now
	meter.delta = 0
	st := meter.state
	meter.mu.Unlock()

	if meter.report != nil {
		meter.report(st)
	}
	return
}
