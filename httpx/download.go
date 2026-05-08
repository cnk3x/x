package httpx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type DonwnloadOptions struct {
	Report       func(state ProgressState) // 进度报告
	SkipIfExists bool                      // 跳过已存在
	Direct       bool                      // 不使用临时文件

	existsState *bool
}

type DownloadOption func(*DonwnloadOptions)

func ProgressReport(report func(state ProgressState)) DownloadOption {
	return func(dOpts *DonwnloadOptions) {
		dOpts.Report = report
	}
}

func SkipExists(existsState *bool) DownloadOption {
	return func(dOpts *DonwnloadOptions) {
		dOpts.SkipIfExists = true
		dOpts.existsState = existsState
	}
}

func (r *Request) Download(ctx context.Context, to string, options ...DownloadOption) error {
	var dOpts DonwnloadOptions
	for _, option := range options {
		option(&dOpts)
	}

	if dOpts.SkipIfExists {
		if exists, err := fileExists(to); exists || err != nil {
			if dOpts.existsState != nil {
				*dOpts.existsState = exists
			}
			return err
		}
	}

	return r.Process(ctx, func(resp *http.Response) (err error) {
		process := func(w *os.File) error {
			if dOpts.Report != nil {
				return SaveResponse(w, resp, dOpts.Report)
			} else {
				return iocopy(w, resp.Body)
			}
		}
		if dOpts.Direct {
			f, err := os.Create(to)
			if err != nil {
				return err
			}
			return cmp.Or(process(f), f.Close())
		}
		return useTempFile(to, process)
	})
}

// FileCheck 存在是非空文件：跳过，存在空文件：删除，不存在：正常返回，打开出错：报错
func fileExists(target string) (exists bool, err error) {
	//检查并跳过存在
	stat, err := os.Stat(target)

	// 路径打开失败
	if err != nil {
		if os.IsNotExist(err) { //不存在，正常反问
			return false, nil
		}
		return false, fmt.Errorf("路径打开失败： %w", err)
	}

	//是文件
	if stat.Mode().IsRegular() {
		if stat.Size() == 0 {
			return false, os.Remove(target) //空文件，删除
		}
		return true, nil //文件非空, 存在跳过
	}

	//存在非文件
	return false, fmt.Errorf("路径冲突： %w", err)
}

func useTempFile(target string, process func(*os.File) error) (err error) {
	if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return
	}

	tempPath := target + ".httpxtmp"
	defer os.Remove(tempPath)

	f, e := os.Create(tempPath)
	if err = e; err != nil {
		return
	}
	if err = cmp.Or(process(f), f.Close()); err != nil {
		return
	}

	if err = os.Remove(target); err != nil && !os.IsNotExist(err) {
		return
	}
	return os.Rename(tempPath, target)
}

const DefaultBufferSize = 512 * 1024

func iocopy(w io.Writer, r io.Reader) error {
	_, err := io.CopyBuffer(w, r, make([]byte, DefaultBufferSize))
	return err
}

var errInvalidWrite = errors.New("invalid write result")

func copyBuffer(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) (err error) {
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}

	var written int64
	for err = ctx.Err(); err == nil; err = ctx.Err() {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	_ = written
	return
}
