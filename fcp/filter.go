package fcp

import (
	"context"
	"io"
	"iter"
	"os"
	"path"
	"strings"
)

// DirSeq 迭代读取文件夹中所有文件信息（非递归）
func DirSeq(ctx context.Context, dir string, excludes []string) iter.Seq2[os.DirEntry, error] {
	return func(yield func(os.DirEntry, error) bool) {
		d, err := os.Open(dir)
		if err != nil {
			yield(nil, err)
			return
		}
		defer d.Close()

		for {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			entries, err := d.ReadDir(500)
			if err != nil {
				if err != io.EOF {
					yield(nil, err)
				}
				return
			}
			for _, info := range entries {
				if err := ctx.Err(); err != nil {
					yield(nil, err)
					return
				}
				if IsExcluded(info.Name(), info.IsDir(), excludes) {
					continue
				}
				if !yield(info, nil) {
					return
				}
			}
		}
	}
}

// IsExcluded 是否排除
//   - name 文件名，仅含名称，有带`/`的 = false
//   - dir 表示文件名是文件夹
//   - exprs 匹配规则
//
// 备注:
//   - 规则为 [path.Glob] 表达式，额外可包含前缀`!`和后缀`/`
//   - `!`表示匹配结果取反(取反规则)，只用来推翻上个匹配成功结果，不表示仅包含
//   - `/`表示只匹配文件夹(文件夹规则)
//   - 文件名级别的规则匹配
//   - 逐行匹配
//   - 忽略格式错误的规则
//   - 取反规则在有效正向规则之前的都忽略。
//   - 正向匹配成功后，忽略后续正向规则，直到找到取反规则，如果取反匹配成功，忽略后续取反规则，直到找到正向匹配，以此类推。
//   - 如果不是文件夹，忽略所有文件夹规则，文件夹参与非文件夹规则匹配
//   - 如果最终结果文件夹匹配成功，整个文件夹含子文件(夹)都被排除
func IsExcluded(name string, dir bool, exprs []string) bool {
	if name == "" {
		return true
	}

	if len(exprs) == 0 {
		return false
	}

	var start bool  // 有效
	var zMatch bool // 正向

	for _, expr := range exprs {
		if expr == "" {
			continue
		}

		expr, invert := strings.CutPrefix(expr, "!")
		if invert && !start {
			continue // 取反规则在有效正向规则之前的都忽略。
		}

		expr, dirOnly := strings.CutSuffix(expr, "/")
		if dirOnly && !dir {
			continue // 如果不是文件夹，忽略所有文件夹规则
		}

		// 状态剪枝：如果当前已经是排除状态且当前又是正向规则，跳过
		if zMatch && !invert {
			continue
		}

		// 状态剪枝：如果当前是包含状态且当前又是取反规则，跳过
		if start && !zMatch && invert {
			continue
		}

		// 匹配执行
		matched, pass := fastMatch(expr, name)
		if pass {
			continue
		}

		start = true

		if matched {
			zMatch = !invert // 正向匹配成功则 zMatch=true; 取反匹配成功则 zMatch=false
		}
	}

	return zMatch
}

func fastMatch(expr, name string) (match, pass bool) {
	if expr == "" || name == "" {
		return false, true // 空规则或空文件名通常不应视为“匹配”或“解析错误”
	}

	// 1. 绝对相等检查 (Zero Allocation)
	if expr == name {
		return true, false
	}

	// 2. 预检通配符
	hasStar := strings.Contains(expr, "*")
	hasQuest := strings.Contains(expr, "?")

	// 3. 处理无通配符的情况：大小写无关相等
	if !hasStar && !hasQuest {
		return strings.EqualFold(expr, name), false
	}

	// 4. 统一转小写进行后续匹配
	expr = strings.ToLower(expr)
	name = strings.ToLower(name)

	// 5. 单个 * 匹配优化
	if hasStar && !hasQuest && ones(expr, '*') {
		if expr == "*" {
			return true, false
		}
		// 长度检查：文件名长度必须 >= 规则长度减去星号
		if len(name) < len(expr)-1 {
			return false, false
		}
		prefix, suffix, _ := strings.Cut(expr, "*")
		return strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix), false
	}

	// 6. 单个 ? 匹配优化
	if hasQuest && !hasStar && ones(expr, '?') {
		if expr == "?" {
			return len(name) == 1, false
		}
		if len(name) != len(expr) {
			return false, false
		}
		prefix, suffix, _ := strings.Cut(expr, "?")
		return strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix), false
	}

	// 7. 复杂情况回退到标准库
	// path.Match 返回 err != nil 表示规则语法错误（如未闭合的 [）
	match, err := path.Match(expr, name)
	return match, err != nil
}

func ones(s string, c byte) bool {
	i := strings.IndexByte(s, c)
	return i != -1 && i == strings.LastIndexByte(s, c)
}
