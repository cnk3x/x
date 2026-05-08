package flagx

import (
	"fmt"
	"slices"
	"strings"
)

// Pairs registers a flag that accepts multiple key-value pairs in the format "key:value".
func Pairs(v *[][2]string, name, short string, def [][2]string, usage string) {
	PairsSet(CommandLine, v, name, short, usage)
}

// PairsSet registers a flag that accepts multiple key-value pairs in the format "key:value" in the given FlagSet.
func PairsSet(fset *FlagSet, v *[][2]string, name, short string, usage string) *FlagItem {
	return fset.VarPF(&pairs{v: v}, name, short, usage)
}

// pairs implements the pflag.SliceValue interface for handling key-value pairs.
type pairs struct {
	v *[][2]string
}

// Append implements the pflag.SliceValue interface by appending a key-value pair in "key:value" format.
func (h *pairs) Append(s string) error {
	if len(s) == 0 {
		return nil
	}
	k, v, ok := strings.Cut(s, ":")
	if !ok {
		return fmt.Errorf("header type expect key:value format, got %s", s)
	}
	k, v = strings.TrimSpace(k), strings.TrimSpace(v)
	*h.v = append(*h.v, [2]string{k, v})
	return nil
}

// GetSlice implements the pflag.SliceValue interface by returning the stored key-value pairs as strings.
func (h *pairs) GetSlice() []string {
	r := make([]string, 0, len(*h.v))
	for i, v := range *h.v {
		r[i] = fmt.Sprintf("%s:%s", v[0], v[1])
	}
	return r
}

// Replace implements the pflag.SliceValue interface by replacing the stored key-value pairs with new ones.
func (h *pairs) Replace(s []string) error {
	*h.v = make([][2]string, len(s))
	for i, v := range s {
		k, v, ok := strings.Cut(v, ":")
		if !ok {
			return fmt.Errorf("header type expect key:value format, got %s", v)
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		(*h.v)[i] = [2]string{k, v}
	}
	return nil
}

// Set implements the pflag.Value interface by setting the value from a multi-line string.
func (h *pairs) Set(s string) error {
	return h.Replace(slices.Collect(strings.Lines(s)))
}

// String implements the pflag.Value interface by returning the stored key-value pairs as a string.
func (h *pairs) String() string {
	return strings.Join(h.GetSlice(), "\n")
}

// Type implements the pflag.Value interface by returning the type name of this value.
func (h *pairs) Type() string {
	return "pairs"
}

// Pair registers a flag that accepts a single key-value pair in the format "key:value".
func Pair(v *[2]string, name, short string, def [][2]string, usage string) {
	PairSet(CommandLine, v, name, short, usage)
}

// PairSet registers a flag that accepts a single key-value pair in the format "key:value" in the given FlagSet.
func PairSet(fset *FlagSet, v *[2]string, name, short string, usage string) *FlagItem {
	return fset.VarPF(&pair{v: v}, name, short, usage)
}

// pair implements the pflag.Value interface for handling a single key-value pair.
type pair struct{ v *[2]string }

// Set implements the pflag.Value interface by setting the key-value pair from a string in "key:value" format.
func (kvp *pair) Set(s string) error {
	k, v, ok := strings.Cut(s, ":")
	if !ok {
		return fmt.Errorf("invalid key-value pair: %q", s)
	}
	kvp.v[0], kvp.v[1] = strings.TrimSpace(k), strings.TrimSpace(v)
	return nil
}

// String implements the pflag.Value interface by returning the key-value pair as a string in "key:value" format.
func (kvp *pair) String() string { return fmt.Sprintf("%s:%s", kvp.v[0], kvp.v[1]) }

// Type implements the pflag.Value interface by returning the type name of this value.
func (kvp *pair) Type() string { return "pair" }