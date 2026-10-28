package gx

import (
	"log/slog"
	"slices"
)

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func ErrDrop[T any](v T, err error) T {
	if err != nil {
		slog.Debug("error ignored", "err", err)
	}
	return v
}

func Iif[T any](c bool, t, f T) T {
	if c {
		return t
	}
	return f
}

func Or[T comparable](a, b T) (r T)       { return Iif(a == r, b, a) }
func In[T comparable](s T, ins ...T) bool { return slices.Contains(ins, s) }

func Some[T any](s []T, predicate func(T) bool) bool {
	return slices.ContainsFunc(s, predicate)
}

func SomeSeq[T any](s func(func(T) bool), predicate func(T) bool) bool {
	for v := range s {
		if predicate(v) {
			return true
		}
	}
	return false
}

func Every[T any](s []T, predicate func(T) bool) bool {
	for _, v := range s {
		if !predicate(v) {
			return false
		}
	}
	return true
}

func EverySeq[T any](s func(func(T) bool), predicate func(T) bool) bool {
	for v := range s {
		if !predicate(v) {
			return false
		}
	}
	return true
}

func MapSeq[T, R any](s func(func(T) bool), convert func(T) R) func(func(R) bool) {
	return func(f func(R) bool) {
		for t := range s {
			if !f(convert(t)) {
				return
			}
		}
	}
}

func Values[T any](s []T) func(func(T) bool) {
	return slices.Values(s)
}

// func IsAllEmpty[T interface{ IsEmpty() bool }](s []T) bool {
// 	for _, v := range s {
// 		if !v.IsEmpty() {
// 			return false
// 		}
// 	}
// 	return true
// }
