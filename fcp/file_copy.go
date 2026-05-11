package fcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
)

// FileCopyOptions 复制文件的选项
type FileCopyOptions struct {
	RemoveSource  bool // 是否删除源文件
	Overwrite     bool // 是否覆盖目标文件
	KeepTime      bool // 保持修改时间
	KeepOwen      bool // 保持所有者
	AutoCreateDir bool // 自动创建文件所在的文件夹（权限默认0755，有需要同步权限时不要设置为true，由上游处理文件夹）

	Report func(delta, copied, total int64) // 已复制增量和总量
}

// FileCopy 原子性复制或移动文件
//   - sourcePath: 源文件路径
//   - targetPath: 目标文件路径
//   - options: 显式传入的配置结构体
//
// 备注:
//   - 采用了原子写入逻辑：先写入 .*.tmp 文件再 Rename，防止目标文件损坏。
//   - 支持跨文件系统移动：当重命名报 syscall.EXDEV 错误时，自动降级为“复制+删除”。
//   - 自动创建路径：目标目录不存在时会自动创建。
func FileCopy(ctx context.Context, sourcePath, targetPath string, options FileCopyOptions) (err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	srcInfo, e := os.Stat(sourcePath)
	if err = e; err != nil {
		err = &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("source stat error: %w", err)}
		return
	}

	// 状态
	var (
		copied atomic.Int64
		total  = srcInfo.Size()
	)

	// 上报
	report := func(delta int64) {
		if options.Report != nil {
			options.Report(delta, copied.Add(delta), total)
		}
	}

	report(0)       // 启动时上报一次
	defer report(0) // 结束时再上报一次

	var targetIsDir bool

	if err = ctx.Err(); err != nil {
		return
	}
	switch dstInfo, de := os.Lstat(targetPath); {
	case de != nil:
		if !os.IsNotExist(de) {
			// 其他错误，比如权限错误等等报错。
			return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: de}
		}
		// 目标不存在
	case dstInfo == nil: // 防御编程，保证下面使用 dstInfo 时不恐慌
		// 正常情况 err == nil 的时候，info != nil 成立
	case dstInfo.IsDir(): // 目标是文件夹
		if !options.Overwrite {
			// 报告存在错误
			return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("target (dir) %w", os.ErrExist)}
		}
		targetIsDir = true
	case dstInfo.Mode()&os.ModeSymlink != 0: // 目标是软链接
		if !options.Overwrite {
			// 报告存在错误
			return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("target (symlink) %w", os.ErrExist)}
		}
	case dstInfo.Mode().IsRegular(): // 常规文件
		// 如果是同文件，跳过复制，直接返回
		if equal, _ := fileEqual(sourcePath, targetPath, srcInfo, dstInfo, false); equal {
			// 因为 fileEqual 不一定完全准确，出于安全考虑，不会对目标文件做任务处理
			return
		}
		if !options.Overwrite {
			// 报告存在错误
			return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("target (regular) %w", os.ErrExist)}
		}
	default:
		// 报告不支持的文件类型错误
		return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("target (%s): %w", dstInfo.Mode().Type(), os.ErrPermission)}
	}

	if options.RemoveSource {
		if err = ctx.Err(); err != nil {
			return
		}
		if err = os.Rename(sourcePath, targetPath); err == nil || !errors.Is(err, syscall.EXDEV) {
			// 没有出错(err == nil)，或者不是跨设备错误(!EXDEV)，
			// 其实 !errors.Is 已经包含了前者条件，为了可读性，增加前者判断
			return
		}
		// 到这里就是跨设备错误，继续 CTR
	}

	if err = ctx.Err(); err != nil {
		return
	}
	sourceFile, e := os.OpenFile(sourcePath, os.O_RDONLY, 0)
	if err = e; err != nil {
		return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("open source file: %w", err)}
	}

	var sourceClosed bool
	defer func() {
		if !sourceClosed {
			sourceFile.Close()
		}
	}()

	dir, name := filepath.Split(targetPath)

	if options.AutoCreateDir {
		if err = ctx.Err(); err != nil {
			return
		}
		if err = os.MkdirAll(dir, 0o755); err != nil {
			return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("create target dir: %w", err)}
		}
	}

	if err = ctx.Err(); err != nil {
		return
	}
	tempFile, e := os.CreateTemp(dir, name+".*.temp")
	if err = e; err != nil {
		return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("create target temp file: %w", err)}
	}

	tempPath := tempFile.Name()

	// 确保清理临时文件
	var renamed bool
	defer func() {
		if !renamed && tempPath != "" {
			os.Remove(tempPath)
		}
	}()

	buf, put := B1M.Get() // 从 sync.Pool 获取缓冲区
	defer put()           // 函数结束时归还

	w := wFunc(func(p []byte) (n int, err error) {
		if err = ctx.Err(); err != nil {
			return
		}
		if n, err = tempFile.Write(p); err != nil {
			return
		}
		report(int64(n))
		return
	})

	if _, err = io.CopyBuffer(w, sourceFile, buf); err != nil {
		tempFile.Close()
		return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("copy to temp: %w", err)}
	}

	if err = tempFile.Sync(); err != nil {
		tempFile.Close()
		return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("sync temp: %w", err)}
	}

	if srcInfo.Mode().Perm() != 0o600 {
		if err = tempFile.Chmod(srcInfo.Mode().Perm()); err != nil {
			tempFile.Close()
			return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("chmod: %w", err)}
		}
	}

	if options.KeepOwen {
		if oe := tempFile.Chown(getOwner(srcInfo)); oe != nil {
			slog.Warn("sync owner error", "path", targetPath, "err", oe)
		}
	}

	if err = tempFile.Close(); err != nil {
		return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("close temp: %w", err)}
	}

	if targetIsDir {
		if err = os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("remove target: %w", err)}
		}
	}

	if err = os.Rename(tempFile.Name(), targetPath); err != nil {
		return &os.LinkError{Op: "atomic_copy", Old: sourcePath, New: targetPath, Err: fmt.Errorf("rename: %w", err)}
	}
	renamed = true // 标记成功，defer 不再尝试 Remove

	if options.KeepTime {
		if te := syncTime(targetPath, srcInfo); te != nil {
			slog.Warn("sync time error", "path", targetPath, "err", te)
		}
	}

	if options.RemoveSource {
		_ = sourceFile.Close()
		sourceClosed = true
		_ = os.Remove(sourcePath) // 容错，允许删除源文件失败。
	}
	return
}

// 文件比对
func FileEqual(srcPath, dstPath string, modTime bool) (equal bool, err error) {
	var srcInfo, dstInfo os.FileInfo
	if srcInfo, err = os.Stat(srcPath); err != nil {
		return
	}
	if dstInfo, err = os.Stat(dstPath); err != nil {
		return
	}
	return fileEqual(srcPath, dstPath, srcInfo, dstInfo, modTime)
}

// 文件比对（内部方法）
func fileEqual(srcPath, dstPath string, srcInfo, dstInfo os.FileInfo, modTime bool) (equal bool, err error) {
	if srcInfo.Size() != dstInfo.Size() {
		return
	}

	if modTime && !srcInfo.ModTime().Equal(dstInfo.ModTime()) {
		return
	}

	err = OpenFor(srcPath, os.O_RDONLY, 0, func(src *os.File) error {
		return OpenFor(dstPath, os.O_RDONLY, 0, func(dst *os.File) error {
			bufEqual := func(b1, b2 []byte, n1, n2 int, e1, e2 error) (bool, error) {
				// 都是 EOF 或者都是 nil, n > 0 才有数据可比
				if n1 > 0 && n1 == n2 && e1 == e2 && (e1 == io.EOF || e1 == nil) && bytes.Equal(b1[:n1], b2[:n2]) {
					return true, e1
				}

				if e1 == io.EOF {
					e1 = nil
				}
				if e1 != nil {
					return false, e1
				}

				if e2 == io.EOF {
					e2 = nil
				}
				return false, e2
			}

			b1, r1 := B32K.Get()
			defer r1()
			b2, r2 := B32K.Get()
			defer r2()

			const blocks = 10
			totalSize := srcInfo.Size()

			if totalSize <= s32K*blocks {
				for {
					n1, e1 := src.Read(b1)
					n2, e2 := dst.Read(b2)
					ok, e := bufEqual(b1, b2, n1, n2, e1, e2)
					if !ok {
						return e
					}
					if e == io.EOF {
						equal = true // 读到末尾且相等
						return nil
					}
				}
			} else {
				// 大文件抽样
				blockSize := totalSize / blocks
				for i := range int64(blocks) {
					var off int64
					switch i {
					case 0:
						off = 0
					case 1:
						off = totalSize - int64(len(b1)) // 确保检查真正的最后 32KB
					default:
						off = blockSize * i
					}

					n1, e1 := src.ReadAt(b1, off)
					n2, e2 := dst.ReadAt(b2, off)
					// ReadAt 读到末尾必然返回 EOF 或 err，这里需兼容处理
					ok, _ := bufEqual(b1, b2, n1, n2, e1, e2)
					if !ok {
						return nil
					}
				}
			}
			equal = true
			return nil
		})
	})
	return
}
