package fcp

import (
	"os"
	"syscall"
	"time"
)

func getOwner(os.FileInfo) (uid, gid int) {
	// Windows上不做所有者处理
	return -1, -1
}

func setChown(*os.File, os.FileInfo, os.FileInfo) error {
	return nil
}

func getTime(info os.FileInfo) (ct, at, wt time.Time) {
	wt = info.ModTime()
	if stat, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		ct = time.Unix(0, stat.CreationTime.Nanoseconds())
		at = time.Unix(0, stat.LastAccessTime.Nanoseconds())
	}
	return
}

func setTime(path string, ct, at, wt time.Time) error {
	if ct.IsZero() {
		if at.IsZero() && wt.IsZero() {
			return nil
		}
		return os.Chtimes(path, at, wt)
	}

	pathPtr, _ := syscall.UTF16PtrFromString(path)
	// 打开文件句柄，需具备写入权限
	handle, err := syscall.CreateFile(pathPtr,
		syscall.FILE_WRITE_ATTRIBUTES,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil, syscall.OPEN_EXISTING, 0, 0)
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(handle)

	var fatp *syscall.Filetime
	if !at.IsZero() {
		fat := syscall.NsecToFiletime(at.UnixNano())
		fatp = &fat
	}

	var fwtp *syscall.Filetime
	if !wt.IsZero() {
		fwt := syscall.NsecToFiletime(wt.UnixNano())
		fwtp = &fwt
	}

	fct := syscall.NsecToFiletime(ct.UnixNano())
	return syscall.SetFileTime(handle, &fct, fatp, fwtp)
}

func syncTime(path string, srcInfo os.FileInfo) error {
	ct, at, wt := getTime(srcInfo)
	return setTime(path, ct, at, wt)
}
