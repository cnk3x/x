package jx

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

type (
	R = json.RawMessage
	N = json.Number
	M = map[string]any

	Raw = R
	Num = N
)

func S[T any](ss ...T) []T { return ss }

func Unmarshal(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func Marshal(v any) ([]byte, error) { return json.Marshal(v) }

func UnmarshalFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if data = bytes.TrimSpace(data); len(data) > 0 {
		return Unmarshal(data, v)
	}
	return nil
}

func MarshalToFile(path string, v any) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func MarshalTo(w io.Writer, v any) error {
	data, err := Marshal(v)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}
