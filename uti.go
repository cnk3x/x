package x

import "cmp"

func MinMax[T cmp.Ordered](v, vMin, vMax, vDef T) (r T) {
	if vMin == vMax && vMin != r {
		return vMin
	}
	if vMin > vMax {
		vMax, vMin = vMin, vMax
	}
	return min(max(cmp.Or(v, vDef), vMin), vMax)
}

func Or[T comparable](v, vDef T) (r T) {
	if v != r {
		return v
	}
	return vDef
}

func Iif[T any](c bool, t, f T) (r T) {
	if c {
		return t
	}
	return f
}

func If[T any](c bool, v T) (x ifs[T]) {
	if c {
		x.ok, x.v = c, v
	}
	return
}

type ifs[T any] struct {
	v  T
	ok bool
}

func (x ifs[T]) ElseIf(c bool, v T) ifs[T] {
	if !x.ok && c {
		x.v = v
	}
	return x
}

func (x ifs[T]) Else(v T) T { return Iif(x.ok, x.v, v) }
