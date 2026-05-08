package urlx

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected RawURL
	}{
		{
			name:  "完整URL",
			input: "https://user:pass@example.com:8080/path/to/resource?query=value#fragment",
			expected: RawURL{
				Scheme:   "https",
				User:     "user",
				Password: "pass",
				Host:     "example.com",
				Port:     "8080",
				Path:     "/path/to/resource",
				Query:    "query=value",
				Fragment: "fragment",
			},
		},
		{
			name:  "无协议URL",
			input: "//example.com/path",
			expected: RawURL{
				Host: "example.com",
				Path: "/path",
			},
		},
		{
			name:  "IPv6 URL",
			input: "http://[2001:db8::1]:8080/path",
			expected: RawURL{
				Scheme: "http",
				Host:   "[2001:db8::1]",
				Port:   "8080",
				Path:   "/path",
			},
		},
		{
			name:  "只有路径",
			input: "/path/to/resource",
			expected: RawURL{
				Path: "/path/to/resource",
			},
		},
		{
			name:  "只有查询参数",
			input: "?query=value",
			expected: RawURL{
				Query: "query=value",
			},
		},
		{
			name:  "只有片段",
			input: "#fragment",
			expected: RawURL{
				Fragment: "fragment",
			},
		},
		{
			name:  "包含转义字符",
			input: "https://user%40example.com:pass%24@example.com/path%20with%20spaces",
			expected: RawURL{
				Scheme:   "https",
				User:     "user@example.com",
				Password: "pass$",
				Host:     "example.com",
				Path:     "/path with spaces",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.input)
			if result != tt.expected {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		input    RawURL
		expected string
	}{
		{
			name: "完整URL",
			input: RawURL{
				Scheme:   "https",
				User:     "user",
				Password: "pass",
				Host:     "example.com",
				Port:     "8080",
				Path:     "/path",
				Query:    "query=value",
				Fragment: "fragment",
			},
			expected: "https://user:pass@example.com:8080/path?query=value#fragment",
		},
		{
			name: "无协议",
			input: RawURL{
				Host: "example.com",
				Path: "/path",
			},
			expected: "//example.com/path",
		},
		{
			name: "只有路径",
			input: RawURL{
				Path: "/path",
			},
			expected: "/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name     string
		input    RawURL
		elem     []string
		expected RawURL
	}{
		{
			name: "相对路径合并",
			input: RawURL{
				Path: "/path/to/file.txt",
			},
			elem: []string{"../other/file.txt"},
			expected: RawURL{
				Path: "/path/other/file.txt",
			},
		},
		{
			name: "绝对路径合并",
			input: RawURL{
				Path: "/path/to/file.txt",
			},
			elem: []string{"/new/path"},
			expected: RawURL{
				Path: "/new/path",
			},
		},
		{
			name: "目录路径合并",
			input: RawURL{
				Path: "/path/to/",
			},
			elem: []string{"file.txt"},
			expected: RawURL{
				Path: "/path/to/file.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.JoinPath(tt.elem...)
			if result.Path != tt.expected.Path {
				t.Errorf("JoinPath(%v) = %v, want %v", tt.elem, result.Path, tt.expected.Path)
			}
		})
	}
}

func TestRemoveQuery(t *testing.T) {
	testURL := RawURL{
		Scheme:   "https",
		Host:     "example.com",
		Path:     "/path",
		Query:    "query=value",
		Fragment: "fragment",
	}

	result := testURL.RemoveQuery()
	if result.Query != "" {
		t.Errorf("RemoveQuery() should clear query, got %q", result.Query)
	}
	if result.Fragment != testURL.Fragment {
		t.Errorf("RemoveQuery() should not affect fragment, got %q", result.Fragment)
	}
}

func TestRemoveFragment(t *testing.T) {
	testURL := RawURL{
		Scheme:   "https",
		Host:     "example.com",
		Path:     "/path",
		Query:    "query=value",
		Fragment: "fragment",
	}

	result := testURL.RemoveFragment()
	if result.Fragment != "" {
		t.Errorf("RemoveFragment() should clear fragment, got %q", result.Fragment)
	}
	if result.Query != testURL.Query {
		t.Errorf("RemoveFragment() should not affect query, got %q", result.Query)
	}
}

func TestSet(t *testing.T) {
	testURL := RawURL{
		Scheme: "http",
		Host:   "example.com",
	}

	result := testURL.Set(
		Scheme("https"),
		Port("443"),
		Path("/new/path"),
	)

	if result.Scheme != "https" {
		t.Errorf("Set(Scheme) should set scheme to https, got %q", result.Scheme)
	}
	if result.Port != "443" {
		t.Errorf("Set(Port) should set port to 443, got %q", result.Port)
	}
	if result.Path != "/new/path" {
		t.Errorf("Set(Path) should set path to /new/path, got %q", result.Path)
	}
	if result.Host != "example.com" {
		t.Errorf("Set should not change host, got %q", result.Host)
	}
}

func TestParseAndStringRoundTrip(t *testing.T) {
	testURLs := []string{
		"https://example.com/path",
		"http://user:pass@example.com:8080/path?query=value#fragment",
		"//example.com/path",
		"/path/to/resource",
	}

	for _, urlStr := range testURLs {
		t.Run(urlStr, func(t *testing.T) {
			parsed := Parse(urlStr)
			reconstructed := parsed.String()
			// 对于包含转义字符的URL，可能需要特殊处理，但这里测试基本功能
			if reconstructed != urlStr {
				t.Errorf("Round trip failed: %q -> %q", urlStr, reconstructed)
			}
		})
	}
}
