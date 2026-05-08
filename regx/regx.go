package regx

import (
	"regexp"
	"sync"
)

var regexCache = &SyncMap[string, *regexp.Regexp]{}

type Options struct {
	Cache bool
}

type Option func(*Options)

func Cache(cache bool) Option {
	return func(o *Options) {
		o.Cache = cache
	}
}

func Compile(expr string, cache bool) (re *regexp.Regexp, err error) {
	if cache {
		r, find := regexCache.Get(expr)
		if re = r; !find {
			if re, err = regexp.Compile(expr); err != nil {
				return
			}
		}
		regexCache.Set(expr, re)
	} else {
		re, err = regexp.Compile(expr)
	}
	return
}

func CompileMust(expr string, cache bool) *regexp.Regexp {
	re, err := Compile(expr, cache)
	if err != nil {
		panic(err)
	}
	return re
}

func Replace(s, find, replace string, options ...Option) string {
	var rOpts Options
	for _, opt := range options {
		opt(&rOpts)
	}

	re, err := Compile(find, rOpts.Cache)
	if err != nil {
		return s
	}
	return re.ReplaceAllString(s, replace)
}

func Match(s, find string, options ...Option) bool {
	var rOpts Options
	for _, opt := range options {
		opt(&rOpts)
	}
	re, err := Compile(find, rOpts.Cache)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

type SyncMap[K, V any] struct{ sync.Map }

func (s *SyncMap[K, V]) Get(key K) (value V, found bool) {
	v, ok := s.Map.Load(key)
	if !ok {
		return
	}
	value, found = v.(V)
	return value, found
}

func (s *SyncMap[K, V]) Set(key K, value V) {
	s.Map.Store(key, value)
}
