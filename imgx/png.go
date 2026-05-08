package imgx

import (
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

var ErrEncoding = errors.New("failed to encode")

var encoderMap = map[string]func(dst io.Writer, img image.Image) error{
	".png":  png.Encode,
	".jpg":  JpegEncode,
	".jpeg": JpegEncode,
}

func ConvertFile(srcPath, dstPath string) error {
	if srcPath == dstPath {
		return nil
	}
	return Convert(srcPath, dstPath, encoderMap[strings.ToLower(filepath.Ext(dstPath))])
}

func ToJpeg(srcPath, dstPath string) error {
	if srcPath == dstPath {
		return nil
	}
	return Convert(srcPath, dstPath, JpegEncode)
}

func ToPng(srcPath, dstPath string) error {
	if srcPath == dstPath {
		return nil
	}
	return Convert(srcPath, dstPath, png.Encode)
}

func Convert(srcPath, dstPath string, imgEncode func(dst io.Writer, img image.Image) error) error {
	if imgEncode == nil {
		return ErrEncoding
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return err
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	return imgEncode(dst, img)
}

func JpegEncode(dst io.Writer, img image.Image) error {
	return jpeg.Encode(dst, img, nil)
}

func ReplaceExt(srcPath, ext string) string {
	return strings.TrimSuffix(srcPath, filepath.Ext(srcPath)) + ext
}
