//go:build linux || darwin

package fcp

import (
	"os"
	"syscall"
	"time"
)

func getOwner(info os.FileInfo) (uid, gid int) {
	if info != nil {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			return int(stat.Uid), int(stat.Gid)
		}
	}
	return -1, -1
}

// if not a root user, return nil to skip chown
func setChown(dstFile *os.File, srcInfo, dstInfo os.FileInfo) error {
	myUid := os.Getuid()
	if myUid != 0 {
		// not a root user, return nil to skip chown
		return nil
	}
	if uid, gid := getOwner(srcInfo); uid > -1 && gid > -1 {
		ouid, ogid := getOwner(dstInfo)
		if ouid != uid || ogid != gid {
			return dstFile.Chown(uid, gid)
		}
	}
	return nil
}

func getTime(info os.FileInfo) (mt time.Time, ct time.Time, at time.Time) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(stat.Mtimespec.Unix()), time.Unix(stat.Ctimespec.Unix()), time.Unix(stat.Atimespec.Unix())
	}
	return info.ModTime(), time.Time{}, time.Time{}
}

func setTime(path string, ct, at, wt time.Time) error {
	if at.IsZero() && wt.IsZero() {
		return nil
	}
	return os.Chtimes(path, at, wt)
}

func syncTime(path string, srcInfo os.FileInfo) error {
	ct, at, wt := getTime(srcInfo)
	return setTime(path, ct, at, wt)
}
