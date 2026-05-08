package flagx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

// File registers a flag that loads data from a file into the provided variable.
// The file content is parsed as JSON by default.
func File[T any](v *T, name, short, def, usage string) {
	FileSet(CommandLine, v, name, short, def, usage)
}

// FileSet registers a flag that loads data from a file into the provided variable in the given FlagSet.
// The file content is parsed as JSON by default.
func FileSet[T any](fset *FlagSet, v *T, name, short, def, usage string) *FlagItem {
	fv := newFileValue(v, def)
	return fset.VarPF(fv, name, short, usage)
}

// FileUnmarshal registers a flag that loads data from a file and unmarshals it using the provided function.
func FileUnmarshal[T any](v *T, name, short, def, usage string, unmarshal func([]byte, any) error) {
	FileUnmarshalSet(CommandLine, v, name, short, def, usage, unmarshal)
}

// FileUnmarshalSet registers a flag that loads data from a file and unmarshals it using the provided function in the given FlagSet.
func FileUnmarshalSet[T any](fset *FlagSet, v *T, name, short, def, usage string, unmarshal func([]byte, any) error) *FlagItem {
	fv := newFileValue(v, def)
	fv.unmarshal = unmarshal
	return fset.VarPF(fv, name, short, usage)
}

// fromFileValue implements the pflag.Value interface for loading values from files.
type fromFileValue[T any] struct {
	val       *T
	isStr     bool
	fp        string
	unmarshal func([]byte, any) error
}

var tStr = reflect.TypeFor[string]()

func newFileValue[T any](v *T, def string) *fromFileValue[T] {
	f := &fromFileValue[T]{val: v, fp: def, isStr: reflect.TypeFor[T]() == tStr}
	if def != "" {
		if e := f.Set(def); e != nil {
			fmt.Printf("[flagx] load default file %s fail: %v\n", def, e)
		}
	}
	return f
}

// Set implements the pflag.Value interface and loads the value from the specified file.
// It includes basic protection against extremely large files.
func (f *fromFileValue[T]) Set(s string) error {
	f.fp = s
	if f.fp == "" {
		return nil
	}

	// 建议：可以使用 os.Stat 检查文件大小，防止读取超大文件导致 OOM
	d, err := os.ReadFile(f.fp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	d = bytes.TrimSpace(d)
	if len(d) == 0 {
		return nil
	}

	if f.isStr {
		if ptr, ok := any(f.val).(*string); ok {
			*ptr = string(d)
			return nil
		}
	}

	if f.unmarshal == nil {
		err = json.Unmarshal(d, f.val)
	} else {
		err = f.unmarshal(d, f.val)
	}

	if err != nil {
		return fmt.Errorf("file %s: %w", f.fp, err)
	}

	return nil
}

// String implements the pflag.Value interface.
func (f *fromFileValue[T]) String() string { return f.fp }

// Type implements the pflag.Value interface.
func (f *fromFileValue[T]) Type() string { return "file" }
