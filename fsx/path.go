package fsx

import (
	"regexp"
	"strings"
	"unicode"
)

// 将正则表达式提取为全局变量，利用 sync.Once 或直接包级初始化实现预编译
var (
	reInvalidChar = regexp.MustCompile(`[<>:"/\\|?*[:cntrl:]_]+`)
	reWhitespace  = regexp.MustCompile(`\s+`)
	reDot         = regexp.MustCompile(`[\s._]+\.[\s._]*`)
	reUnderscore  = regexp.MustCompile(`\s*_\s*`)
)

// CleanFileName 清理文件名，移除非法字符、控制字符、下划线、连续空白字符、连续点号、下划线前后的多余空格
//
//	可选参数 keepHeadDot：是否保留首尾的点号（默认保留）
//	返回清理后的文件名, 如果文件名为空, 则返回空字符串
func CleanFileName(name string) string {
	if name == "" {
		return ""
	}
	//替换非法字符、控制字符及下划线为单一下划线
	name = reInvalidChar.ReplaceAllString(name, "_")
	//连续空白字符合并为一个空格
	name = reWhitespace.ReplaceAllString(name, " ")
	//连续点号和空格和下划线合并为一个点号
	name = reDot.ReplaceAllString(name, ".")
	//去除下划线前后的多余空格
	name = reUnderscore.ReplaceAllString(name, "_")
	//首尾的空格和点号
	name = strings.TrimFunc(name, notTail)

	return name
}

func notTail(r rune) bool { return unicode.IsSpace(r) || r == '.' || r == '_' }
