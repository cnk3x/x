package arrx

type I interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

type U interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type F interface {
	~float32 | ~float64
}

type N interface {
	I | U | F
}
