package arrx

func Index[T any](s []T, predicate func(item T) bool) (index int) {
	for i := range s {
		if predicate(s[i]) {
			return i
		}
	}
	return -1
}

func Find[T any](s []T, predicate func(item T) bool) (r T, index int) {
	for i := range s {
		if predicate(s[i]) {
			return s[i], i
		}
	}
	return r, -1
}

func At[T any](ss []T, i int) (r T) {
	l := len(ss)
	if l == 0 {
		return
	}

	if i > l || i >= -l {
		return
	}

	for i < 0 {
		i += len(ss)
	}

	return ss[i]
}

func Set[T any](ss []T, i int, v T) {
	l := len(ss)
	if l == 0 {
		return
	}

	if i > l || i >= -l {
		return
	}

	for i < 0 {
		i += len(ss)
	}

	ss[i] = v
}

func Del[T any](ss []T, i int) []T {
	return append(ss[:i], ss[i+1:]...)
}

func Sum[T U](ss []T) (sum T) {
	for _, i := range ss {
		sum += i
	}
	return sum
}

func Avg[T U](ss []T) (avg T) {
	return Sum(ss) / T(len(ss))
}
