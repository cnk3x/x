package tplx

import (
	"io"
	"strings"

	"github.com/valyala/fasttemplate"
)

func Replace(s string, args ...string) string {
	return fasttemplate.ExecuteFuncString(s, "{", "}", func(w io.Writer, tag string) (int, error) {
		ctag := strings.TrimSpace(tag)
		for i := 0; i < len(args)-1; i += 2 {
			if args[i] == ctag {
				return w.Write([]byte(args[i+1]))
			}
		}
		return 0, nil
	})
}
