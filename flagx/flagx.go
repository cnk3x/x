// Package flagx provides enhanced command-line flag parsing capabilities beyond
// the standard 'flag' library, supporting struct tags, environment variables,
// configuration files, and key-value pairs.
package flagx

import (
	"cmp"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/pflag"
)

type (
	// FlagItem represents a single flag item in the FlagSet.
	FlagItem = pflag.Flag
	// FlagSet represents a set of flags that can be managed together.
	FlagSet = pflag.FlagSet
)

var (
	// Parse parses the command-line arguments.
	Parse = pflag.Parse
	// CommandLine is the default FlagSet that manages command-line flags.
	CommandLine = pflag.CommandLine
)

var (
	// Description holds the description of the application shown in help.
	Description = ""
	// Version holds the version of the application shown in help.
	Version = ""
)

var appName = filepath.Base(os.Args[0])

func init() {
	Init(CommandLine, appName)
	pflag.ErrHelp = fmt.Errorf("\nstart with %s [...OPTIONS]", appName)
	pflag.Usage = func() {
		w := CommandLine.Output()
		fmt.Fprintf(w, "%s", appName)
		if Version != "" {
			fmt.Fprintf(w, " - %s", Version)
		}
		fmt.Fprintf(w, "\n")

		if Description != "" {
			fmt.Fprintf(w, "  %s\n", Description)
		}
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "Usage:\n")
		fmt.Fprintf(w, "  %s [OPTIONS]\n\n", appName)
		fmt.Fprintf(w, "OPTIONS:\n")
		pflag.PrintDefaults()
	}
}

// Usage prints the usage information for the command-line flags.
func Usage() {
	pflag.Usage()
}

// Name returns the name of the application.
func Name() string { return appName }

// Init initializes a FlagSet with the given name.
func Init(fSet *FlagSet, name string) {
	fSet.Init(name, pflag.ExitOnError)
	fSet.SortFlags = false // 不对标志进行排序
}

// Var adds a command-line flag based on the value type.
func Var[T flagType](val *T, name, shorthand, usage string, env ...string) {
	VarSet(CommandLine, val, name, shorthand, usage, env...)
}

// VarSet adds a command-line flag based on the value type to the specified FlagSet.
func VarSet[T flagType](fset *FlagSet, val *T, name, shorthand, usage string, env ...string) (f *FlagItem, err error) {
	return addToSet(fset, val, name, shorthand, usage, env...)
}

// Struct defines command-line flags from fields of a struct.
func Struct[T any](pStruct *T) {
	StructSet(CommandLine, pStruct, false)
}

// StructSet defines command-line flags from fields of a struct in the specified FlagSet.
func StructSet[T any](fset *FlagSet, pStruct *T, errExit bool) (err error) {
	return structToSet(fset, nil, nil, nil, pStruct, errExit)
}

func addToSet(fset *FlagSet, val any, name, shorthand, usage string, env ...string) (f *FlagItem, err error) {
	if fset.ShorthandLookup(shorthand) != nil {
		fmt.Printf("[warn] flag %s's shorthand %s is already exists\n", name, shorthand)
		shorthand = ""
	}

	if fset.Lookup(name) != nil {
		fmt.Printf("[warn] flag %s already exists\n", name)
		return
	}

	switch x := val.(type) {
	case *[2]string:
		PairSet(fset, x, name, shorthand, usage)
	case *[][2]string:
		PairsSet(fset, x, name, shorthand, usage)
	case *time.Duration:
		fset.DurationVarP(x, name, shorthand, *x, usage)
	case *net.IP:
		fset.IPVarP(x, name, shorthand, *x, usage)
	case *net.IPNet:
		fset.IPNetVarP(x, name, shorthand, *x, usage)
	case *string:
		fset.StringVarP(x, name, shorthand, *x, usage)
	case *int:
		fset.IntVarP(x, name, shorthand, *x, usage)
	case *int8:
		fset.Int8VarP(x, name, shorthand, *x, usage)
	case *int16:
		fset.Int16VarP(x, name, shorthand, *x, usage)
	case *int32:
		fset.Int32VarP(x, name, shorthand, *x, usage)
	case *int64:
		fset.Int64VarP(x, name, shorthand, *x, usage)
	case *uint:
		fset.UintVarP(x, name, shorthand, *x, usage)
	case *uint8:
		fset.Uint8VarP(x, name, shorthand, *x, usage)
	case *uint16:
		fset.Uint16VarP(x, name, shorthand, *x, usage)
	case *uint32:
		fset.Uint32VarP(x, name, shorthand, *x, usage)
	case *uint64:
		fset.Uint64VarP(x, name, shorthand, *x, usage)
	case *float32:
		fset.Float32VarP(x, name, shorthand, *x, usage)
	case *float64:
		fset.Float64VarP(x, name, shorthand, *x, usage)
	case *bool:
		fset.BoolVarP(x, name, shorthand, *x, usage)
	case *[]time.Duration:
		fset.DurationSliceVarP(x, name, shorthand, *x, usage)
	case *[]net.IP:
		fset.IPSliceVarP(x, name, shorthand, *x, usage)
	case *[]net.IPNet:
		fset.IPNetSliceVarP(x, name, shorthand, *x, usage)
	case *[]string:
		fset.StringSliceVarP(x, name, shorthand, *x, usage)
	case *[]int:
		fset.IntSliceVarP(x, name, shorthand, *x, usage)
	case *[]int32:
		fset.Int32SliceVarP(x, name, shorthand, *x, usage)
	case *[]int64:
		fset.Int64SliceVarP(x, name, shorthand, *x, usage)
	case *[]uint:
		fset.UintSliceVarP(x, name, shorthand, *x, usage)
	case *[]float32:
		fset.Float32SliceVarP(x, name, shorthand, *x, usage)
	case *[]float64:
		fset.Float64SliceVarP(x, name, shorthand, *x, usage)
	case *[]bool:
		fset.BoolSliceVarP(x, name, shorthand, *x, usage)
	default:
		err = fmt.Errorf("%s type %v(%T) not support", name, x, x)
		return
	}

	f = fset.Lookup(name)

	// 检查是否是被弃用的标记
	if matches := reDeprecated.FindStringSubmatch(f.Usage); len(matches) > 1 {
		f.Usage = f.Usage[:len(f.Usage)-len(matches[0])]
		f.Deprecated = matches[1]
		if f.Deprecated == "" {
			f.Deprecated = "deprecated"
		}
	}

	// 如果该标志有对应的环境变量，则处理环境变量值
	if len(env) > 0 {
		if f.Usage != "" {
			f.Usage += " "
		}
		f.Usage = fmt.Sprintf("%s[%s]", f.Usage, strings.Join(env, ", "))
		if s := getEnv(env); s != "" {
			if e := f.Value.Set(s); e != nil {
				fmt.Fprintf(os.Stderr, "WARN: set flag `%s` value `%s` from environ: %s\n", f.Name, s, e)
			}
		}
	}

	return
}

func structToSet(fset *FlagSet, prefixes []string, usagePrefixes []string, envKeyPrefix []string, pStruct any, errExit bool) (err error) {
	var rv reflect.Value

	// 1. 统一转换为 reflect.Value
	if v, ok := pStruct.(reflect.Value); ok {
		rv = v
	} else {
		rv = reflect.ValueOf(pStruct)
	}

	// 2. 指针合法性校验
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("flagx: pStruct must be a pointer, got %v", rv.Kind())
	}
	if rv.IsNil() {
		return fmt.Errorf("flagx: pStruct is nil")
	}

	// 3. 核心校验：剥离一层指针后必须是结构体
	// 这一步直接封死了多重指针 (**Struct) 和 非结构体指针 (*int)
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		if rv.Kind() == reflect.Pointer {
			return fmt.Errorf("flagx: multi-level pointers are not allowed")
		}
		return fmt.Errorf("flagx: pointer must point to a struct, got %v", rv.Kind())
	}

	// 4. 可寻址性二次确认（防御性编程）
	if !rv.CanAddr() {
		return fmt.Errorf("flagx: struct is not addressable")
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)

		// 跳过非导出字段
		if !f.IsExported() {
			continue
		}

		name, short, usage, envKeys := getTags(f)
		if name == "-" {
			continue
		}

		vField := rv.Field(i)

		// 检查是否是嵌套结构体
		isStruct, isPointer, isNil := checkStruct(f.Type, vField)
		if isStruct {
			// --- 递归前初始化逻辑 ---
			if isPointer && isNil {
				// 如果字段是 *Struct 且为 nil，在递归前初始化它
				vField = fieldInit(f.Type, vField)
			}

			if len(envKeys) > 0 {
				envKeys = concat(envKeyPrefix, at(envKeys, 0))
			} else {
				envKeys = envKeyPrefix
			}

			if name == "" && !f.Anonymous {
				name = lower(f.Name)
			}

			err = structToSet(fset,
				concat(prefixes, name),
				concat(usagePrefixes, usage),
				envKeys,
				vField.Addr(),
				errExit,
			)
		} else {
			// 跳过不支持类型的字段
			if !allowType(f.Type, false) {
				fmt.Fprintf(os.Stderr, "WARN: field %q type %q not support\n", f.Name, f.Type)
				continue
			}

			if name == "" {
				name = lower(f.Name)
			}

			if usage == "" {
				tn := cmp.Or(rt.Name(), rt.String())
				usage = fmt.Sprintf("%s.%s", tn, f.Name)
			}

			name = strings.Join(concat(prefixes, name), ".")
			usage = strings.Join(concat(usagePrefixes, usage), "")
			envKeys = joins(envKeyPrefix, envKeys, "_")

			if isPointer && isNil {
				vField = fieldInit(f.Type, vField)
			}
			_, err = addToSet(fset, vField.Addr().Interface(), name, short, usage, envKeys...)
		}

		if err != nil && errExit {
			return err
		}
	}
	return nil
}

func checkStruct(t reflect.Type, v reflect.Value) (isStruct, isPointer, isNil bool) {
	isPointer = t.Kind() == reflect.Pointer
	if isPointer {
		t = t.Elem()
	}
	isStruct = t.Kind() == reflect.Struct

	isNil = IsNilSafe(v)
	return
}

type flagType interface {
	time.Duration | net.IP | net.IPNet | [2]string |
		string |
		int | int8 | int16 | int32 | int64 |
		uint | uint8 | uint16 | uint32 | uint64 |
		float32 | float64 |
		bool |
		[]time.Duration | []net.IP | []net.IPNet |
		[]string |
		[]int | []int32 | []int64 |
		[]uint | []uint32 | []uint64 |
		[]float32 | []float64 |
		[]bool | [][2]string
}

var (
	tDuration  = reflect.TypeFor[time.Duration]()
	tIP        = reflect.TypeFor[net.IP]()
	tIPNet     = reflect.TypeFor[net.IPNet]()
	tDurations = reflect.TypeFor[[]time.Duration]()
	tIPs       = reflect.TypeFor[[]net.IP]()
	tIPNets    = reflect.TypeFor[[]net.IPNet]()

	tKvPair  = reflect.TypeFor[[2]string]()
	tKvPairs = reflect.TypeFor[[][2]string]()
)

// allowType: 检查类型是否在 [flagType] 中，切片只允许单层切片
func allowType(t reflect.Type, inSlice bool) bool {
	// 1. 剥离所有指针，拿到基类型
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	// 2. 特殊 Type 匹配
	switch t {
	case tDuration, tIP, tIPNet, tKvPair:
		return true
	case tDurations, tIPs, tIPNets, tKvPairs:
		return !inSlice // 如果已经在 slice 里了，就不允许双重 slice
	}

	// 3. Kind 匹配
	switch kind := t.Kind(); kind {
	case reflect.String,
		reflect.Int, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		return true
	case reflect.Int8, reflect.Int16, reflect.Uint8, reflect.Uint16:
		return !inSlice // pflag 不支持这些基本类型的 SliceVar
	case reflect.Slice:
		if inSlice {
			return false // 禁止 [][]T
		}
		return allowType(t.Elem(), true) //递归切片类型
	default:
		return false
	}
}

// IsNilSafe checks if a reflect.Value is nil.
//
//	Only pointers, slices, maps, channels, functions or interfaces can call IsNil.
//	Returns false for other types.
func IsNilSafe(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}

// 初始化指针字段
func fieldInit(t reflect.Type, v reflect.Value) reflect.Value {
	// 只有当 v 是指针且为 nil 时才需要初始化
	if t.Kind() == reflect.Pointer && IsNilSafe(v) {
		// 创建一个指向基础类型的实例
		// 例如：t 是 **int, 则 t.Elem() 是 *int
		newVal := reflect.New(t.Elem())
		v.Set(newVal)

		// 递归处理：初始化刚刚创建出来的这一层指针
		return fieldInit(t.Elem(), v.Elem())
	}
	return v
}

// getEnv gets the first non-empty value from environment variables.
//   - Checks the provided environment variable keys in order and returns the first existing value
func getEnv(keys []string) (s string) {
	if len(keys) > 0 {
		for _, e := range keys {
			if s = os.Getenv(e); s != "" {
				return
			}
		}
	}
	return ""
}

// 匹配弃用信息
var reDeprecated = regexp.MustCompile(`\s*\*\*DEPRECATED\*\*\s*(.*)$`)

// lower 将驼峰命名转换为下划线分隔的小写形式
//   - 例如：MaxConnection -> max_connection
func lower(s string) string {
	var b strings.Builder
	var prevUp bool
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i != 0 && !prevUp {
				b.WriteRune('_')
			}
			prevUp = true
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
			prevUp = false
		}
	}
	return b.String()
}

func getTags(f reflect.StructField) (name, short, usage string, envKeys []string) {
	flag := f.Tag.Get("flag")
	if flag == "-" {
		name = "-"
		return
	}

	flags := cleanSplit(flag, ",", 3, false)
	name = flags[0]
	short = cmp.Or(f.Tag.Get("short"), flags[1])
	usage = cmp.Or(f.Tag.Get("usage"), flags[2])
	envKeys = cleanSplit(f.Tag.Get("env"), " ", -1, true)
	return
}

func cleanSplit(s string, sep string, n int, removeEmpty bool) []string {
	r := strings.SplitN(s, sep, n)
	for i := range r {
		r[i] = strings.TrimSpace(r[i])
	}

	if removeEmpty {
		var x int
		for _, v := range r {
			if v != "" {
				r[x], x = v, x+1
			}
		}
		r = r[:x]
	} else {
		if n > 0 {
			for len(r) < n {
				r = append(r, "")
			}
		}
	}
	return r
}

// concat merges strings into a slice, removing empty strings.
//   - If s is an empty string, it is not added to ss
//   - If s is a non-empty string, it is added to ss
func concat(ss []string, s string) []string {
	if s = strings.TrimSpace(s); s != "" {
		return append(ss, s)
	}
	return ss
}

// joins merges strings into a slice, removing empty strings
//   - If s is an empty string, it is not added to ss
//   - If s is a non-empty string, it is added to ss
func joins(prefixes []string, ss []string, sep string) (r []string) {
	for _, s := range ss {
		r = append(r, strings.Join(concat(prefixes, s), sep))
	}
	return
}

func at[T any](ss []T, i int) (r T) {
	l := len(ss)
	if l == 0 {
		return
	}

	if i < 0 {
		i += len(ss)
	}

	if i > l || i < 0 {
		return
	}

	return ss[i]
}
