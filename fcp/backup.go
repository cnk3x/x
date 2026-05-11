package fcp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
)

type BackupState struct {
	options  BackupOptions
	progress BackupProgress
	eg       *errgroup.Group
}

func (s *BackupState) handleError(source, target string, err error) error {
	if err == nil {
		return nil
	}
	if s.options.ErrReport != nil {
		s.options.ErrReport(source, target, err.Error())
	}
	if !s.options.ErrContinue {
		return err
	}
	return nil
}

func (s *BackupState) ProgressReport() {
	if s.options.ProgressReport != nil {
		s.options.ProgressReport(s.Progress())
	}
}

type BackupProgress struct {
	TotalSize   int64
	TotalCount  int64
	CopiedSize  int64
	CoppedCount int64
}

func (s BackupState) Progress() BackupProgress {
	return BackupProgress{
		TotalSize:   s.progress.TotalSize,
		TotalCount:  s.progress.TotalCount,
		CopiedSize:  s.progress.CopiedSize,
		CoppedCount: s.progress.CoppedCount,
	}
}

type BackupOptions struct {
	RemoveSource bool // 是否删除源文件
	Overwrite    bool // 是否覆盖目标文件
	KeepTime     bool // 保持时间
	KeepOwen     bool // 保持所有者

	MaxParallel int  // 最大并行数
	ErrContinue bool // 出错继续

	// 排除
	//  - 文件名级别的Glob匹配
	//  - 跳过格式错误的Glob表达式
	//  - 增加前缀!表示匹配结果取反
	//  - 增加末尾`/`表示只匹配文件夹
	//  - 逐行匹配
	//  - 取反匹配只用来推翻上一个匹配成功结果，不表示仅包含，第一行的取反匹配忽略。
	//  - 直到成功和没有取反匹配，有取反匹配直到失败。
	//  - 如果文件夹匹配成功，整个文件夹含子文件(夹)都被排除
	Excludes []string

	ErrReport      func(source, target, err string)
	ProgressReport func(progress BackupProgress)
}

func Backup(ctx context.Context, source, target string, options BackupOptions) (err error) {
	stat, err := os.Stat(source)
	if err != nil {
		return err
	}

	state := &BackupState{options: options}

	g, gc := errgroup.WithContext(ctx)
	if options.MaxParallel > 0 {
		g.SetLimit(min(options.MaxParallel, runtime.NumCPU()*2))
	}
	state.eg = g

	if stat.IsDir() {
		g.Go(handleBackupDir(gc, source, target, state))
	} else {
		g.Go(handleBackupFile(gc, source, target, state))
	}

	return g.Wait()
}

func handleBackupDir(ctx context.Context, source, target string, state *BackupState) func() error {
	return func() error {
		for entriy, err := range DirSeq(ctx, source, state.options.Excludes) {
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return err
				}
			}

			if err = ctx.Err(); err != nil {
				return err
			}

			if err := state.handleError(source, target, err); err != nil {
				return err
			}

			subSource := filepath.Join(source, entriy.Name())
			subTarget := filepath.Join(target, entriy.Name())

			if entriy.IsDir() {
				state.eg.Go(handleBackupDir(ctx, subSource, subTarget, state))
				continue
			}

			info, err := entriy.Info()
			if err != nil {
				return err
			}

			if info.Mode().IsRegular() {
				atomic.AddInt64(&state.progress.TotalCount, 1)
				atomic.AddInt64(&state.progress.TotalSize, info.Size())
				state.ProgressReport()
				state.eg.Go(handleBackupFile(ctx, subSource, subTarget, state))
				continue
			}

			if err := state.handleError(source, target, err); err != nil {
				return err
			}
		}
		return nil
	}
}

func handleBackupFile(ctx context.Context, source, target string, state *BackupState) func() error {
	return func() error {
		err := FileCopy(ctx, source, target, FileCopyOptions{
			RemoveSource:  state.options.RemoveSource,
			Overwrite:     state.options.Overwrite,
			KeepTime:      state.options.KeepTime,
			KeepOwen:      state.options.KeepOwen,
			AutoCreateDir: false,
			Report:        func(delta, _, _ int64) { atomic.AddInt64(&state.progress.CopiedSize, delta) },
		})

		if err == nil {
			atomic.AddInt64(&state.progress.CoppedCount, 1)
			state.ProgressReport()
			return nil
		}
		return state.handleError(source, target, err)
	}
}
