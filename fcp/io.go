package fcp

import "os"

// 打开文件，并在操作完成后自动关闭
func OpenFor(filePath string, mode int, perm os.FileMode, process func(*os.File) error) (err error) {
	f, err := os.OpenFile(filePath, mode, perm)
	if err != nil {
		return err
	}

	if err = process(f); err != nil {
		f.Close()
		return err
	}

	if err = f.Close(); err != nil {
		return
	}

	return nil
}
