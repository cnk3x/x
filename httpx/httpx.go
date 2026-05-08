package httpx

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

var jar http.CookieJar

func init() {
	jar, _ = cookiejar.New(nil)
}

type Request struct {
	Method             string
	URL                string
	Headers            [][2]string
	Body               func(ctx context.Context) (io.ReadCloser, string, error)
	RaiseStatus        bool
	TryTimes           int
	Timeout            time.Duration
	Proxy              string
	ProxyDisabled      bool
	InsecureSkipVerify bool
	Jar                http.CookieJar
	BeforeTry          func(err error, tryTime int, sleep time.Duration)
	Dump               bool
}

type Process func(*http.Response) error

func New(url string) *Request {
	return &Request{
		Method: "GET",
		URL:    url,
		Headers: [][2]string{
			{"User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0"},
			{"Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6,zh-TW;q=0.5"},
			{"Accept", "*/*"},
		},
		TryTimes: 4,
	}
}

func NewUrl(url *url.URL) *Request { return New(url.String()) }

func (r *Request) With(options ...Option) *Request {
	for _, option := range options {
		option(r)
	}
	return r
}

func (r *Request) Do(ctx context.Context) (resp *http.Response, err error) {
	var (
		body        io.ReadCloser
		contentType string
	)

	if r.Body != nil {
		if body, contentType, err = r.Body(ctx); err != nil {
			return
		}
		if body != nil {
			defer body.Close()
		}
	}

	req, e := http.NewRequestWithContext(ctx, r.Method, r.URL, io.NopCloser(body))
	if err = e; err != nil {
		return
	}

	for _, h := range r.Headers {
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

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	transport := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: r.InsecureSkipVerify},
		TLSNextProto:          make(map[string]func(string, *tls.Conn) http.RoundTripper),
	}

	if !r.ProxyDisabled {
		transport.Proxy = http.ProxyFromEnvironment
		if r.Proxy != "" {
			proxyURL, e := url.Parse(r.Proxy)
			if err = e; err != nil {
				return
			}
			transport.Proxy = http.ProxyURL(proxyURL)
			if r.Dump {
				fmt.Println("使用代理：", proxyURL.String())
			}
		}
	}

	client := &http.Client{Timeout: r.Timeout, Transport: transport, Jar: r.Jar}

	if client.Jar == nil {
		client.Jar = jar
	}

	if resp, err = client.Do(req); err != nil {
		return
	}

	if r.Dump {
		in, _ := httputil.DumpRequest(resp.Request, false)
		fmt.Println("请求:")
		fmt.Println(strings.TrimSpace(string(in)))
		out, _ := httputil.DumpResponse(resp, true)
		fmt.Println()
		fmt.Println("响应:")
		fmt.Println(strings.TrimSpace(string(out)))
	}
	return
}

func (r *Request) processResponse(ctx context.Context, process Process) (err error) {
	resp, e := r.Do(ctx)
	if err = e; err != nil {
		return
	}
	defer resp.Body.Close()

	if r.RaiseStatus && resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			err = fmt.Errorf("需要登录: %d", resp.StatusCode)
		case http.StatusForbidden:
			err = fmt.Errorf("权限不足: %d", resp.StatusCode)
		default:
			err = fmt.Errorf("status code %d not ok", resp.StatusCode)
		}
		d, _ := io.ReadAll(resp.Body)
		return &StatusError{Status: resp.StatusCode, Raw: d}
	}

	return process(resp)
}

func (r *Request) Process(ctx context.Context, process Process) (err error) {
	const retryDelay = time.Second
	for attempt := range r.TryTimes + 1 {
		if attempt > 0 {
			delay := max(min(retryDelay<<attempt, time.Second*10), time.Second)
			if r.BeforeTry != nil {
				r.BeforeTry(err, attempt, delay)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		if err = r.processResponse(ctx, process); shouldRetry(err) {
			continue
		}
		break
	}
	return
}

func (r *Request) GetBytes(ctx context.Context) (data []byte, err error) {
	err = r.Process(ctx, func(resp *http.Response) (err error) {
		data, err = io.ReadAll(resp.Body)
		return
	})
	return
}

func (r *Request) Exec(ctx context.Context) (err error) {
	return r.Process(ctx, func(r *http.Response) error {
		return iocopy(io.Discard, r.Body)
	})
}
