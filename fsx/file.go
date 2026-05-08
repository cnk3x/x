package fsx

import (
	"io"
	"os"
	"path/filepath"
)

// RenameFile 单个文件重命名，自动覆盖目标（如果目标存在且为单文件或者空文件夹）
func RenameFile(from, to string) error {
	if err := CreateDir(filepath.Dir(to)); err != nil {
		return err
	}

	err := Eon(os.Stat(to))
	if err == nil {
		err = os.Remove(to) //存在, 尝试删除（非递归）
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return os.Rename(from, to)
}

// CreateDir 创建目录，如果路径存在且是零文件，尝试删除后再创建
func CreateDir(path string) error {
	info, err := os.Stat(path)

	if err != nil {
		if !os.IsNotExist(err) {
			//其他错误，比如权限
			return err
		}

		// 不存在创建
		err = os.MkdirAll(path, 0755)
	} else {
		//非文件夹，删除
		if !info.IsDir() {
			//如果文件大小不为0，则返回存在错误
			if info.Size() > 0 {
				return os.ErrExist
			}
			// 尝试删除
			err = os.Remove(path)
		}
	}

	return err
}

func UseTempFile(dstPath string, process func(tempFile *os.File) error) (err error) {
	if err = os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return
	}
	tempPath := dstPath + ".tmp"
	defer os.Remove(tempPath)

	if err = Open(tempPath, process, Writeable()); err != nil {
		return
	}

	return RenameFile(tempPath, dstPath)
}

func Eon[V, E any](_ V, e E) E { return e }
func Von[V, E any](v V, _ E) V { return v }

func DirIsEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
