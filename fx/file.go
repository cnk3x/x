package fx

import (
	"io"
	"io/fs"
	"os"

	"github.com/cnk3x/x/gx"
)

type FileProcess func(*os.File) error

// CreateTemp like [os.CreateTemp], auto close after write, return temp file path
func CreateTemp(dir, pattern string, write FileProcess) (string, error) {
	tempFile, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()
	if err := write(tempFile); err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}

// CreateInTemp like [os.CreateTemp], dir is [os.TempDir], auto close after write, return temp file path
func CreateInTemp(pattern string, write FileProcess) (string, error) {
	return CreateTemp(os.TempDir(), pattern, write)
}

// Create like [os.OpenFile], auto close after useFile
func OpenFile(name string, flag int, perm fs.FileMode, useFile FileProcess) error {
	f, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return err
	}
	return gx.Or(useFile(f), f.Close())
}

// Create like [os.Open], auto close after readFile
func Open(name string, readFile FileProcess) error {
	return OpenFile(name, os.O_RDONLY, 0, readFile)
}

// Create like [os.Create], auto close after writeFile
func Create(name string, writeFile FileProcess) error {
	return OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666, writeFile)
}

func Process[W io.Writer](v any, process func(f W, v any) error) FileProcess {
	return func(f *os.File) error {
		return process(io.Writer(f).(W), v)
	}
}

func Blob(data []byte) FileProcess {
	return func(f *os.File) error { _, err := f.Write(data); return err }
}
