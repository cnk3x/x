package httpx

import (
	"context"
	"io"
	"net/http"
	"time"
)

type Request struct {
	Method  string
	URL     string
	Headers [][2]string
	Body    func(ctx context.Context) (io.ReadCloser, string, error)

	Try    TryOptions
	Client ClientOptions

	RaiseStatus bool
}

type ClientOptions struct {
	Proxy      ProxyOptions
	Timeout    time.Duration
	Jar        http.CookieJar
	SkipVerify bool
}

type ProxyOptions struct {
	Disabled bool
	Server   string
	NoProxy  string
}

type TryOptions struct {
	// MaxAttempt 最大尝试次数（包含首次请求）
	// 例如：MaxAttempt=3 表示最多执行 3 次请求（1次初始 + 2次重试）
	MaxAttempt    int
	BeforeAttempt func(err error, attempt int, sleep time.Duration)
	MaxRetryDelay time.Duration // 默认 15s
	MinRetryDelay time.Duration // 默认 1s
}

type Process func(*http.Response) error

type contextKey string

const statusContextKey contextKey = "httpx-status"

// New 创建一个请求
func New(url string) *Request {
	return &Request{
		Method: "GET",
		URL:    url,
		Headers: [][2]string{
			{"User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0"},
			{"Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6,zh-TW;q=0.5"},
			{"Accept", "*/*"},
		},
		Try: TryOptions{
			MaxAttempt: 5,
		},
	}
}

// Get 创建一个 GET 请求
func Get(url string) *Request {
	return New(url)
}

// Post 创建一个 POST 请求
func Post(url string) *Request {
	return New(url).With(Method("POST"))
}

// Put 创建一个 PUT 请求
func Put(url string) *Request {
	return New(url).With(Method("PUT"))
}

// Patch 创建一个 PATCH 请求
func Patch(url string) *Request {
	return New(url).With(Method("PATCH"))
}

// Delete 创建一个 DELETE 请求
func Delete(url string) *Request {
	return New(url).With(Method("DELETE"))
}

// Head 创建一个 HEAD 请求
func Head(url string) *Request {
	return New(url).With(Method("HEAD"))
}
