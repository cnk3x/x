package httpx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestMaxAttempt_Basic 测试基本的 MaxAttempt 行为
func TestMaxAttempt_Basic(t *testing.T) {
	attemptCount := 0

	// 创建一个总是返回错误的测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(
			MaxAttempt(3),
			RaiseStatus(),
		).
		Exec(ctx)

	if err == nil {
		t.Error("期望得到错误，但得到了 nil")
	}

	if attemptCount != 3 {
		t.Errorf("期望执行 3 次尝试，实际执行了 %d 次", attemptCount)
	}
}

// TestMaxAttempt_SuccessOnRetry 测试在重试后成功
func TestMaxAttempt_SuccessOnRetry(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(
			MaxAttempt(5),
			RaiseStatus(),
		).
		Exec(ctx)

	if err != nil {
		t.Errorf("期望成功，但得到错误: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("期望执行 3 次尝试（2次失败 + 1次成功），实际执行了 %d 次", attemptCount)
	}
}

// TestMaxAttempt_DefaultValue 测试默认 MaxAttempt 值
func TestMaxAttempt_DefaultValue(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	// 不设置 MaxAttempt，使用默认值 5
	err := New(server.URL).
		With(RaiseStatus()).
		Exec(ctx)

	if err == nil {
		t.Error("期望得到错误，但得到了 nil")
	}

	if attemptCount != 5 {
		t.Errorf("期望执行 5 次尝试（默认值），实际执行了 %d 次", attemptCount)
	}
}

// TestMaxAttempt_ExactlyOne 测试 MaxAttempt=1 只执行一次
func TestMaxAttempt_ExactlyOne(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(
			MaxAttempt(1),
			RaiseStatus(),
		).
		Exec(ctx)

	if err == nil {
		t.Error("期望得到错误，但得到了 nil")
	}

	if attemptCount != 1 {
		t.Errorf("期望执行 1 次尝试，实际执行了 %d 次", attemptCount)
	}
}

// TestBeforeAttempt_Callback 测试 BeforeAttempt 回调函数
func TestBeforeAttempt_Callback(t *testing.T) {
	attemptCount := 0
	callbackCalls := []struct {
		attempt int
		delay   time.Duration
	}{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(
			MaxAttempt(3),
			RaiseStatus(),
			BeforeAttempt(func(err error, attempt int, delay time.Duration) {
				callbackCalls = append(callbackCalls, struct {
					attempt int
					delay   time.Duration
				}{attempt, delay})
			}),
		).
		Exec(ctx)

	if err == nil {
		t.Error("期望得到错误，但得到了 nil")
	}

	// 应该有 2 次回调（第 2 次和第 3 次尝试前）
	if len(callbackCalls) != 2 {
		t.Errorf("期望 2 次回调调用，实际调用了 %d 次", len(callbackCalls))
	}

	// 验证回调参数
	for i, call := range callbackCalls {
		expectedAttempt := i + 1 // attempt 从 1 开始
		if call.attempt != expectedAttempt {
			t.Errorf("第 %d 次回调：期望 attempt=%d，实际 attempt=%d", i+1, expectedAttempt, call.attempt)
		}
		if call.delay <= 0 {
			t.Errorf("第 %d 次回调：期望 delay>0，实际 delay=%v", i+1, call.delay)
		}
	}
}

// TestRetryDelay_ExponentialBackoff 测试指数退避延迟
func TestRetryDelay_ExponentialBackoff(t *testing.T) {
	delays := []time.Duration{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	_ = New(server.URL).
		With(
			MaxAttempt(4),
			RaiseStatus(),
			BeforeAttempt(func(err error, attempt int, delay time.Duration) {
				delays = append(delays, delay)
			}),
		).
		Exec(ctx)

	// 应该有 3 次重试（attempt=1,2,3）
	if len(delays) != 3 {
		t.Fatalf("期望 3 次延迟记录，实际记录了 %d 次", len(delays))
	}

	// 验证指数退避：2s, 4s, 8s（受 maxDelay=15s 限制）
	expectedDelays := []time.Duration{
		2 * time.Second, // 2^1
		4 * time.Second, // 2^2
		8 * time.Second, // 2^3
	}

	for i, expected := range expectedDelays {
		if delays[i] != expected {
			t.Errorf("第 %d 次重试：期望延迟 %v，实际延迟 %v", i+1, expected, delays[i])
		}
	}
}

// TestRetryDelay_WithMaxLimit 测试延迟不超过最大值
func TestRetryDelay_WithMaxLimit(t *testing.T) {
	delays := []time.Duration{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	_ = New(server.URL).
		With(
			MaxAttempt(6),
			RaiseStatus(),
			MaxRetryDelay(5*time.Second), // 设置较小的最大值
			BeforeAttempt(func(err error, attempt int, delay time.Duration) {
				delays = append(delays, delay)
			}),
		).
		Exec(ctx)

	// 验证所有延迟都不超过最大值
	maxDelay := 5 * time.Second
	for i, delay := range delays {
		if delay > maxDelay {
			t.Errorf("第 %d 次重试：延迟 %v 超过了最大值 %v", i+1, delay, maxDelay)
		}
	}
}

// TestRetryDelay_WithMinLimit 测试延迟不低于最小值
func TestRetryDelay_WithMinLimit(t *testing.T) {
	delays := []time.Duration{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	_ = New(server.URL).
		With(
			MaxAttempt(3),
			RaiseStatus(),
			MinRetryDelay(2*time.Second), // 设置较大的最小值
			BeforeAttempt(func(err error, attempt int, delay time.Duration) {
				delays = append(delays, delay)
			}),
		).
		Exec(ctx)

	// 验证所有延迟都不低于最小值
	minDelay := 2 * time.Second
	for i, delay := range delays {
		if delay < minDelay {
			t.Errorf("第 %d 次重试：延迟 %v 低于最小值 %v", i+1, delay, minDelay)
		}
	}
}

// TestRetryAfter_Header 测试 Retry-After 头的解析和使用
func TestRetryAfter_Header(t *testing.T) {
	attemptCount := 0
	lastDelay := time.Duration(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			w.Header().Set("Retry-After", "3") // 3 秒
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(
			MaxAttempt(5),
			RaiseStatus(),
			BeforeAttempt(func(err error, attempt int, delay time.Duration) {
				lastDelay = delay
			}),
		).
		Exec(ctx)

	if err != nil {
		t.Errorf("期望成功，但得到错误: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("期望执行 3 次尝试，实际执行了 %d 次", attemptCount)
	}

	// 最后一次重试的延迟应该是 Retry-After + 1s = 4s
	expectedDelay := 4 * time.Second
	if lastDelay != expectedDelay {
		t.Errorf("期望最后延迟为 %v，实际为 %v", expectedDelay, lastDelay)
	}
}

// TestRetryAfter_ExceedsMax 测试 Retry-After 超过最大值时放弃重试
func TestRetryAfter_ExceedsMax(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.Header().Set("Retry-After", "20") // 20 秒，超过默认的 15s 上限
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(
			MaxAttempt(5),
			RaiseStatus(),
		).
		Exec(ctx)

	if err == nil {
		t.Error("期望得到错误，但得到了 nil")
	}

	// 应该只执行 1 次，因为 Retry-After 超过上限直接返回
	if attemptCount != 1 {
		t.Errorf("期望执行 1 次尝试，实际执行了 %d 次", attemptCount)
	}
}

// TestRetryAfter_MultipleStatusCodes 测试多种状态码的 Retry-After 支持
func TestRetryAfter_MultipleStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"429 Too Many Requests", http.StatusTooManyRequests},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
		{"504 Gateway Timeout", http.StatusGatewayTimeout},
		{"425 Too Early", 425},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			attemptCount := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount++
				if attemptCount < 2 {
					w.Header().Set("Retry-After", "1")
					w.WriteHeader(tc.statusCode)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			ctx := context.Background()
			err := New(server.URL).
				With(
					MaxAttempt(3),
					RaiseStatus(),
				).
				Exec(ctx)

			if err != nil {
				t.Errorf("%s: 期望成功，但得到错误: %v", tc.name, err)
			}

			if attemptCount != 2 {
				t.Errorf("%s: 期望执行 2 次尝试，实际执行了 %d 次", tc.name, attemptCount)
			}
		})
	}
}

// TestContextCancellation 测试 context 取消
func TestContextCancellation(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// 在第 2 次尝试前取消
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := New(server.URL).
		With(
			MaxAttempt(5),
			RaiseStatus(),
		).
		Exec(ctx)

	if err != context.Canceled {
		t.Errorf("期望 context.Canceled 错误，实际得到: %v", err)
	}

	// 应该至少执行了 1 次
	if attemptCount < 1 {
		t.Errorf("期望至少执行 1 次尝试，实际执行了 %d 次", attemptCount)
	}
}

// TestCustomRetryDelays 测试自定义重试延迟配置
func TestCustomRetryDelays(t *testing.T) {
	delays := []time.Duration{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	_ = New(server.URL).
		With(
			MaxAttempt(4),
			RaiseStatus(),
			MaxRetryDelay(3*time.Second),
			MinRetryDelay(500*time.Millisecond),
			BeforeAttempt(func(err error, attempt int, delay time.Duration) {
				delays = append(delays, delay)
			}),
		).
		Exec(ctx)

	// 验证延迟在自定义范围内
	for i, delay := range delays {
		if delay < 500*time.Millisecond {
			t.Errorf("第 %d 次重试：延迟 %v 低于最小值 500ms", i+1, delay)
		}
		if delay > 3*time.Second {
			t.Errorf("第 %d 次重试：延迟 %v 超过最大值 3s", i+1, delay)
		}
	}
}

// TestNoRetryOnSuccess 测试成功时不重试
func TestNoRetryOnSuccess(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(
			MaxAttempt(5),
			RaiseStatus(),
		).
		Exec(ctx)

	if err != nil {
		t.Errorf("期望成功，但得到错误: %v", err)
	}

	if attemptCount != 1 {
		t.Errorf("期望执行 1 次尝试（成功不重试），实际执行了 %d 次", attemptCount)
	}
}

// TestRaiseStatusFalse 测试 RaiseStatus=false 时的行为
func TestRaiseStatusFalse(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx := context.Background()
	err := New(server.URL).
		With(MaxAttempt(3)).
		// 不设置 RaiseStatus()，默认为 false
		Exec(ctx)

	// RaiseStatus=false 时，即使状态码不是 200 也不会重试
	if err != nil {
		t.Errorf("RaiseStatus=false 时期望无错误，但得到: %v", err)
	}

	if attemptCount != 1 {
		t.Errorf("RaiseStatus=false 时期望执行 1 次尝试，实际执行了 %d 次", attemptCount)
	}
}
