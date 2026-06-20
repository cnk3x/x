package httpx

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
)

// Bytes 读取响应体并解析为字节数组
func Bytes(read func(data []byte) error) Process {
	return func(resp *http.Response) (err error) {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}
		err = read(data)
		return
	}
}

// JSON 读取响应体并解析为 JSON
func JSON(v any) Process {
	return Bytes(func(data []byte) error {
		if len(data) == 0 {
			return nil
		}
		return json.Unmarshal(data, v)
	})
}

// XML 读取响应体并解析为 XML
func XML(v any) Process {
	return Bytes(func(data []byte) error {
		if len(data) == 0 {
			return nil
		}
		return xml.Unmarshal(data, v)
	})
}
