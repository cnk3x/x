package httpx

import (
	"cmp"
	"context"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func (r *Request) With(options ...Option) *Request {
	for _, option := range options {
		option(r)
	}
	return r
}

// Option is a request option.
type Option func(*Request)

// Options 设置请求选项。
func Options(options ...Option) Option { return func(r *Request) { r.With(options...) } }

// Method 设置请求方法。
func Method(method string) Option { return func(r *Request) { r.Method = method } }

// RaiseStatus 设置请求是否 raises 状态码。
func RaiseStatus() Option { return func(r *Request) { r.RaiseStatus = true } }

// HeaderSet 设置请求头。
func HeaderSet(key, value string) Option {
	return func(r *Request) {
		r.Headers = append(r.Headers, [2]string{key, value})
	}
}

// HeaderSets 设置请求头，支持多行
//
// 例如：
//
//	```
//	User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.90 Safari/537.36
//	Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9
//	Accept-Language: zh-CN,zh;q=0.9
//	```
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

// UserAgent 设置请求头。
func UserAgent(userAgent string) Option {
	return HeaderSet("User-Agent", userAgent)
}

// Referer 设置请求头。
func Referer(referer string) Option {
	return HeaderSet("Referer", referer)
}

// Cookies 设置请求头。
func Cookies(cookies string) Option {
	return HeaderSet("Cookie", cookies)
}

// Query 设置查询参数。
func Query(params url.Values) Option {
	return func(r *Request) {
		u, _ := url.Parse(r.URL)
		q := u.Query()
		for k, vs := range params {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
		r.URL = u.String()
	}
}

// Body 设置请求体。
func Body(body string) func(ctx context.Context) (io.ReadCloser, string, error) {
	return func(ctx context.Context) (io.ReadCloser, string, error) {
		var contentType string
		bType, content, _ := strings.Cut(body, ":")
		switch bType {
		case "file":
			ext := filepath.Ext(content)
			contentType = cmp.Or(mime.TypeByExtension(ext), "application/octet-stream")
			f, err := os.Open(filepath.Clean(content))
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
}

func BodyFile(path string) Option {
	return func(r *Request) {
		r.Body = func(ctx context.Context) (io.ReadCloser, string, error) {
			ext := filepath.Ext(path)
			contentType := cmp.Or(mime.TypeByExtension(ext), "application/octet-stream")
			f, err := os.Open(filepath.Clean(path)) // 防止路径遍历
			return f, contentType, err
		}
	}
}

// Form 设置请求体。
func Form(form url.Values) func(ctx context.Context) (io.ReadCloser, string, error) {
	return func(ctx context.Context) (io.ReadCloser, string, error) {
		isMultiPart := false
		for k, v := range form {
			if len(v) > 1 || strings.Contains(k, "@") {
				isMultiPart = true
				break
			}
		}

		if !isMultiPart {
			return io.NopCloser(strings.NewReader(form.Encode())), "application/x-www-form-urlencoded", nil
		}

		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)

		go func() {
			for k, vs := range form {
				for _, v := range vs {
					if err := ctx.Err(); err != nil {
						pw.CloseWithError(ctx.Err())
						return
					}

					var err error
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
			pw.CloseWithError(writer.Close())
		}()

		return pr, writer.FormDataContentType(), nil
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

	buf := b64k.Get()
	defer b64k.Put(buf)
	_, err = copyBuffer(ctx, fw, f, buf)
	return
}
