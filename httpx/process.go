package httpx

import (
	"cmp"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	// maxTryDelay 最大重试延迟时间
	maxDelay = 15 * time.Second
	// minTryDelay 最小重试延迟时间
	minDelay = 1 * time.Second
)

// process 处理请求(单次处理)
func (r *Request) process(ctx context.Context, process Process) (err error) {
	var req *http.Request
	if req, err = buildRequest(ctx, r); err != nil {
		return
	}

	var client *http.Client
	if client, err = buildClient(r.Client); err != nil {
		return
	}

	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	resp.Request = resp.Request.WithContext(context.WithValue(ctx, statusContextKey, resp.StatusCode))

	if r.RaiseStatus && resp.StatusCode != http.StatusOK {
		d, _ := io.ReadAll(resp.Body)
		se := &StatusError{Status: resp.StatusCode, Raw: d}
		// 如果是 429，尝试解析 Retry-After 头
		if resp.StatusCode == http.StatusTooManyRequests ||
			resp.StatusCode == http.StatusServiceUnavailable ||
			resp.StatusCode == http.StatusGatewayTimeout ||
			resp.StatusCode == http.StatusTooEarly {
			if retryAfter := parseRetryAfter(resp); retryAfter > 0 {
				se.RetryAfter = retryAfter
			}
		}
		return se
	}

	return process(resp)
}

// Process 处理请求(含重试处理)
func (r *Request) Process(ctx context.Context, process Process) (err error) {
	maxRetryDelay := cmp.Or(r.Try.MaxRetryDelay, maxDelay)
	minRetryDelay := cmp.Or(r.Try.MinRetryDelay, minDelay)
	maxAttempt := cmp.Or(r.Try.MaxAttempt, 1) // 默认至少执行 1 次

	for attempt := range maxAttempt {
		if attempt > 0 {
			var delay time.Duration

			// 如果是 429 错误且有 Retry-After，优先使用
			var se *StatusError
			if errors.As(err, &se) && se.RetryAfter > 0 {
				// RetryAfter + 1s: 避免刚好在边界
				if delay = se.RetryAfter + time.Second; delay > maxRetryDelay {
					// 如果超过最大等待时间，放弃重试
					return err
				}
			} else {
				// 使用指数退避，最小1秒，最大15秒
				delay = time.Second * time.Duration(1<<uint(attempt))
				delay = min(delay, maxRetryDelay)
			}

			delay = max(delay, minRetryDelay)
			if r.Try.BeforeAttempt != nil {
				r.Try.BeforeAttempt(err, attempt, delay)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// 等待完成，继续执行请求
			}
		}

		if err = r.process(ctx, process); shouldRetry(err) {
			continue
		}
		break
	}
	return
}

// Exec 执行请求，丢弃响应体
func (r *Request) Exec(ctx context.Context) (err error) {
	return r.Process(ctx, func(r *http.Response) error {
		_, err := io.Copy(io.Discard, r.Body)
		return err
	})
}

// GetBytes 获取响应体字节数组
func (r *Request) GetBytes(ctx context.Context) (data []byte, err error) {
	err = r.Process(ctx, func(resp *http.Response) (err error) {
		data, err = io.ReadAll(resp.Body)
		return
	})
	return
}

// buildClient 构建 HTTP 客户端
func buildClient(r ClientOptions) (*http.Client, error) {
	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{},
	}

	if r.SkipVerify {
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	if !r.Proxy.Disabled {
		transport.Proxy = http.ProxyFromEnvironment
		if r.Proxy.Server != "" {
			proxyURL, err := url.Parse(r.Proxy.Server)
			if err != nil {
				return nil, err
			}
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{Timeout: r.Timeout, Transport: transport, Jar: r.Jar}, nil
}

// buildRequest 构建 HTTP 请求
func buildRequest(ctx context.Context, r *Request) (req *http.Request, err error) {
	var (
		body        io.ReadCloser
		contentType string
	)

	if r.Body != nil {
		if body, contentType, err = r.Body(ctx); err != nil {
			return
		}

		// http.NewRequestWithContext 会正确处理 io.ReadCloser 的生命周期

		// if body != nil {
		// 	srcBody := body
		// 	defer srcBody.Close()
		// }

		// body = io.NopCloser(body)
	}

	if req, err = http.NewRequestWithContext(ctx, r.Method, r.URL, body); err != nil {
		return
	}

	headersApply(req, r.Headers)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return req, err
}

// headersApply 应用请求头
func headersApply(req *http.Request, headers [][2]string) {
	for _, h := range headers {
		k, v := h[0], h[1]

		if k == "" {
			continue
		}

		if k == "-" {
			if v != "" {
				req.Header.Del(v)
			}
			continue
		}

		if v == "" {
			continue
		}

		switch k[0] {
		case '+':
			req.Header.Add(k[1:], v)
		case '-':
			req.Header.Del(k[1:])
		default:
			req.Header.Set(k, v)
		}
	}
}
