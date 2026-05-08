package pdfx

import (
	"os"
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// 合并多个PDF成一个PDF
func PDFs(outFile string, inFiles []string) (err error) {
	if err = os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return
	}
	return api.MergeCreateFile(inFiles, outFile, false, nil)
}

// 将多个图片转换成PDF，每个图片一页
func Images(outFile string, inFiles []string) (err error) {
	if err = os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return
	}
	return api.ImportImagesFile(inFiles, outFile, nil, nil)
}
