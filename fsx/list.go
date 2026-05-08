package fsx

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ListFiles 遍历 sources 中的路径，返回文件列表
// sources: 需要遍历的文件或目录路径列表
// recursive: 是否递归遍历子目录
// onError: 错误处理回调函数，当遇到无法访问的路径时触发
func ListFiles(sources []string, recursive bool, onError func(error) error) (files []string, err error) {
	// 如果未提供错误处理函数，使用默认处理：打印警告并继续（返回 nil）
	if onError == nil {
		onError = func(err error) error {
			slog.Warn(fmt.Sprintf("⚠️ 警告: %v\n", err))
			return nil
		}
	}

	for _, src := range sources {
		// 1. 转换为绝对路径
		absPath, e := filepath.Abs(src)
		if e != nil {
			if err = onError(fmt.Errorf("无法获取绝对路径 %q: %w", src, e)); err != nil {
				return
			}
			continue
		}

		// 2. 获取路径信息
		info, e := os.Stat(absPath)
		if e != nil {
			if err = onError(fmt.Errorf("无法访问路径 %q: %w", absPath, e)); err != nil {
				return
			}
			continue
		}

		// 3. 如果是文件，直接加入
		if !info.IsDir() {
			files = append(files, absPath)
			continue
		}

		// 4. 目录遍历
		if recursive {
			// --- 策略 A: 递归遍历 ---
			walkErr := filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					// 这里的 err 通常是权限问题
					return onError(err)
				}
				if !d.IsDir() {
					files = append(files, path)
				}
				return nil
			})
			if walkErr != nil {
				// WalkDir 本身出错（极少见，除非根目录被删）
				if err = onError(fmt.Errorf("遍历目录 %q 失败: %w", absPath, walkErr)); err != nil {
					return
				}
			}
		} else {
			// --- 策略 B: 仅遍历一级 ---
			entries, e := os.ReadDir(absPath)
			if e != nil {
				if err = onError(fmt.Errorf("读取目录 %q: %w", absPath, e)); err != nil {
					return
				}
				continue
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					files = append(files, filepath.Join(absPath, entry.Name()))
				}
			}
		}
	}

	return
}

// ByLength 排序规则：路径长度降序（深层在前），长度相同时按字典序升序
func ByLength(reverse bool) SortFunc {
	return func(a, b string) int {
		if reverse {
			a, b = b, a
		}
		switch l := len(a) - len(b); {
		case l > 0:
			return 1
		case l < 0:
			return -1
		default:
			return 0
		}
	}
}

func ByName(reverse bool) SortFunc {
	if reverse {
		return func(a, b string) int { return strings.Compare(b, a) }
	}
	return func(a, b string) int { return strings.Compare(a, b) }
}

type SortFunc func(a, b string) int

func Sorts(sorts ...SortFunc) SortFunc {
	return func(a, b string) int {
		for _, s := range sorts {
			if r := s(a, b); r != 0 {
				return r
			}
		}
		return 0
	}
}

func Split(path string) (dir, name, ext string) {
	dir, name = filepath.Split(path)
	ext = filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)
	return
}
