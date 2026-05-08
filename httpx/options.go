package httpx

import (
	"cmp"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Option func(*Request)

func Options(options ...Option) Option { return func(r *Request) { r.With(options...) } }

func Method(method string) Option { return func(r *Request) { r.Method = method } }

func RaiseStatus() Option { return func(r *Request) { r.RaiseStatus = true } }

func TryTimes(tryTimes int) Option { return func(r *Request) { r.TryTimes = tryTimes } }

func Insecure(insecure bool) Option { return func(r *Request) { r.InsecureSkipVerify = insecure } }

func Timeout(timeout time.Duration) Option { return func(r *Request) { r.Timeout = timeout } }

func Proxy(proxy string) Option {
	return func(r *Request) {
		if i, _ := strconv.Atoi(proxy); i > 0 && i <= 65535 {
			r.Proxy = "http://127.0.0.1:" + proxy
			return
		}

		u, err := url.Parse(proxy)
		if err != nil {
			r.Proxy = proxy
			return
		}

		if u.Scheme == "" {
			u.Scheme = "http"
		}

		if host, port := u.Hostname(), u.Port(); host == "" || port == "" {
			if port == "" {
				switch u.Scheme {
				case "https":
					port = "443"
				case "http":
					port = "80"
				case "socks", "socks5":
					port = "1080"
				default:
					return
				}
			}
			if host == "" {
				host = "127.0.0.1"
			}
			u.Host = fmt.Sprintf("%s:%s", host, port)
		}

		r.Proxy = u.String()
	}
}

func ProxyDisabled() Option {
	return func(r *Request) { r.ProxyDisabled = true }
}

func BeforeTry(beforeTry func(err error, tryTime int, sleep time.Duration)) Option {
	return func(r *Request) { r.BeforeTry = beforeTry }
}

func Dump() Option {
	return func(r *Request) { r.Dump = true }
}

func HeaderSet(key, value string) Option {
	return func(r *Request) {
		r.Headers = append(r.Headers, [2]string{key, value})
	}
}

func HeaderSets(headers string) Option {
	return func(r *Request) {
		for line := range strings.Lines(headers) {
			if k, v, ok := strings.Cut(line, ":"); ok {
				k, v = strings.TrimSpace(k), strings.TrimSpace(v)
				r.Headers = append(r.Headers, [2]string{k, v})
			}
		}
	}
}

func UserAgent(userAgent string) Option {
	return HeaderSet("User-Agent", userAgent)
}

func Referer(referer string) Option {
	return HeaderSet("Referer", referer)
}

func Cookies(cookies string) Option {
	return HeaderSet("Cookie", cookies)
}

func CookieJar(jar http.CookieJar) Option {
	return func(r *Request) { r.Jar = jar }
}

func Body(body string) Option {
	return func(r *Request) {
		r.Body = func(ctx context.Context) (io.ReadCloser, string, error) {
			var contentType string
			bType, content, _ := strings.Cut(body, ":")
			switch bType {
			case "file":
				ext := filepath.Ext(content)
				contentType = cmp.Or(mime.TypeByExtension(ext), "application/octet-stream")
				f, err := os.Open(content)
				return f, contentType, err
			case "base64":
				b := base64.NewDecoder(base64.StdEncoding, strings.NewReader(content))
				return io.NopCloser(b), "application/octet-stream", nil
			case "json":
				return io.NopCloser(strings.NewReader(content)), "application/json", nil
			case "form":
				return io.NopCloser(strings.NewReader(content)), "application/x-www-form-urlencoded", nil
			case "xml":
				return io.NopCloser(strings.NewReader(content)), "application/xml", nil
			case "html":
				return io.NopCloser(strings.NewReader(content)), "text/html", nil
			default:
				return io.NopCloser(strings.NewReader(body)), contentType, nil
			}
		}

		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
			r.Method = http.MethodPost
		}
	}
}

func Form(form url.Values) Option {
	return func(r *Request) {
		r.Body = func(ctx context.Context) (body io.ReadCloser, contentType string, err error) {
			isMultiPart := false
			for k, v := range form {
				if len(v) > 1 || strings.Contains(k, "@") {
					isMultiPart = true
					break
				}
			}

			if !isMultiPart {
				body = io.NopCloser(strings.NewReader(form.Encode()))
				contentType = "application/x-www-form-urlencoded"
				return
			}

			pr, pw := io.Pipe()
			writer := multipart.NewWriter(pw)

			go func() {
				for k, vs := range form {
					for _, v := range vs {
						if strings.HasPrefix(v, "@") {
							err = writeFileField(ctx, writer, k, v[1:])
						} else {
							err = writer.WriteField(k, v)
						}
						if err == nil {
							err = ctx.Err()
						}
						if err != nil {
							pw.CloseWithError(err)
							return
						}
					}
				}
				if err == nil {
					err = ctx.Err()
				}
				pw.CloseWithError(writer.Close())
			}()

			return pr, writer.FormDataContentType(), nil
		}

		if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch {
			r.Method = http.MethodPost
		}
	}
}

func writeFileField(ctx context.Context, writer *multipart.Writer, fieldName, filePath string) (err error) {
	f, e := os.Open(filePath)
	if err = e; err != nil {
		return
	}
	defer f.Close()

	fw, e := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err = e; err != nil {
		return
	}

	err = copyBuffer(ctx, fw, f, make([]byte, 512*1024))
	return
}
