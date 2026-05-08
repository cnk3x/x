package fsx

import (
	"io"
	"os"
)

func Fcp(srcPath, dstPath string) error {
	return Open(srcPath, WriteToFile(dstPath))
}

func Open(path string, process func(r *os.File) error, options ...FileOpenOption) error {
	var opts FileOpenOptions
	for _, option := range options {
		option(&opts)
	}

	f, err := os.OpenFile(path, opts.Flag, opts.Perm)
	if err != nil {
		return err
	}
	defer f.Close()
	return process(f)
}

func WriteToFile(dstPath string) func(src *os.File) error {
	return func(src *os.File) error {
		return Open(dstPath, From(src), Writeable(os.O_TRUNC))
	}
}

func From(src *os.File) func(dst *os.File) error {
	return func(dst *os.File) error { _, err := io.Copy(dst, src); return err }
}

func To(dst *os.File) func(src *os.File) error {
	return func(src *os.File) error { _, err := io.Copy(dst, src); return err }
}

type FileOpenOptions struct {
	Flag int
	Perm os.FileMode
}

type FileOpenOption func(*FileOpenOptions)

func Writeable(appendFlag ...int) FileOpenOption {
	return func(opts *FileOpenOptions) {
		opts.Flag = os.O_RDWR | os.O_CREATE
		if len(appendFlag) == 0 {
			appendFlag = []int{os.O_TRUNC}
		}
		for _, flag := range appendFlag {
			opts.Flag |= flag
		}
	}
}

func Flag(flag int) FileOpenOption {
	return func(opts *FileOpenOptions) {
		opts.Flag = flag
	}
}
