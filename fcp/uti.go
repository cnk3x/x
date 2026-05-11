package fcp

type wFunc func(p []byte) (n int, err error)

func (w wFunc) Write(p []byte) (n int, err error) { return w(p) }
