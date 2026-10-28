package ex

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cnk3x/x/gx"
)

var (
	ExeBin  string
	ExeDir  string
	ExeName string
	ExeExt  string
)

func init() {
	ExeBin = gx.Must(os.Executable())
	ExeDir = filepath.Dir(ExeBin)
	ExeName = filepath.Base(ExeBin)
	ExeExt = filepath.Ext(ExeName)
	if gx.In(strings.ToLower(ExeExt), ".com", ".exe", ".bat", ".cmd") {
		ExeName = ExeName[:len(ExeName)-len(ExeExt)]
	}
}
