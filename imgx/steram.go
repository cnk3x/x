package imgx

import (
	"image/jpeg"
	"io"
	"sync"

	"golang.org/x/image/webp"
)

func Webp2JpegStream(webpStream io.Reader) (jepgStream io.ReadCloser) {
	pr, pw := io.Pipe()
	var wg sync.WaitGroup
	wg.Go(func() {
		img, e := webp.Decode(webpStream)
		if e != nil {
			pw.CloseWithError(e)
			return
		}
		if e := jpeg.Encode(pw, img, nil); e != nil {
			pw.CloseWithError(e)
		} else {
			pw.Close()
		}
	})

	return &Rc{
		Reader: pr,
		close:  func() error { wg.Wait(); return pr.Close() },
	}
}

type Rc struct {
	io.Reader
	close func() error
}

func (rc Rc) Close() error { return rc.close() }
