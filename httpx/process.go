package httpx

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
)

func JSON(v any) Process {
	return func(resp *http.Response) (err error) {
		data, e := io.ReadAll(resp.Body)
		if err = e; err != nil {
			return
		}
		if len(data) == 0 {
			return
		}
		err = json.Unmarshal(data, v)
		return
	}
}

func XML(v any) Process {
	return func(resp *http.Response) (err error) {
		data, e := io.ReadAll(resp.Body)
		if err = e; err != nil {
			return
		}
		if len(data) == 0 {
			return
		}
		err = xml.Unmarshal(data, v)
		return
	}
}
