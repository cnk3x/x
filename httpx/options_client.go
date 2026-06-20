package httpx

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Try 设置重试选项
func Try(tryOpt TryOptions) Option {
	return func(r *Request) { r.Try = tryOpt }
}

// BeforeAttempt 设置重试前的回调函数
func BeforeAttempt(beforeAttempt func(err error, attempt int, sleep time.Duration)) Option {
	return func(r *Request) { r.Try.BeforeAttempt = beforeAttempt }
}

// MaxAttempt 设置最大尝试次数（包含首次请求）
func MaxAttempt(maxAttempt int) Option {
	return func(r *Request) { r.Try.MaxAttempt = maxAttempt }
}

// MaxRetryDelay 设置最大重试延迟时间
func MaxRetryDelay(delay time.Duration) Option {
	return func(r *Request) { r.Try.MaxRetryDelay = delay }
}

// MinRetryDelay 设置最小重试延迟时间
func MinRetryDelay(delay time.Duration) Option {
	return func(r *Request) { r.Try.MinRetryDelay = delay }
}

// Insecure 设置是否跳过 SSL 证书验证
func Insecure(insecure bool) Option { return func(r *Request) { r.Client.SkipVerify = insecure } }

// Timeout 设置请求超时时间
func Timeout(timeout time.Duration) Option { return func(r *Request) { r.Client.Timeout = timeout } }

// Proxy 设置 HTTP 代理
//   - 当输入是纯端口号时添加 localhost
//   - 如果缺少 scheme，添加默认 http
func Proxy(proxy string) Option {
	return func(r *Request) {
		if proxy == "" {
			return
		}

		// 仅当输入是纯端口号时才添加 localhost
		if port, err := strconv.Atoi(proxy); err == nil && port > 0 && port <= 65535 {
			r.Client.Proxy.Server = fmt.Sprintf("http://127.0.0.1:%d", port)
			return
		}

		// 尝试解析为 URL
		u, err := url.Parse(proxy)
		if err != nil {
			r.Client.Proxy.Server = proxy
			return
		}

		// 如果缺少 scheme，添加默认 http
		if u.Scheme == "" {
			u.Scheme = "http"
		}

		r.Client.Proxy.Server = u.String()
	}
}

// ProxyDisabled 禁用 HTTP 代理
func ProxyDisabled() Option {
	return func(r *Request) { r.Client.Proxy.Disabled = true }
}

// CookieJar 设置 CookieJar
func CookieJar(jar http.CookieJar) Option {
	return func(r *Request) { r.Client.Jar = jar }
}
