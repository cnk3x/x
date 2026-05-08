package urlx

import (
	"bytes"
	"net/url"
	"path"
	"strings"
)

// 解析RawURL
func Parse(s string) (u RawURL) {
	s, u.Fragment, _ = strings.Cut(s, "#")
	s, u.Query, _ = strings.Cut(s, "?")

	if strings.HasPrefix(s, "//") {
		s = s[2:]
	} else if strings.Contains(s, "://") {
		u.Scheme, s, _ = strings.Cut(s, "://")
	}

	if strings.HasPrefix(s, "/") {
		u.Path, s = s, ""
	} else {
		s, u.Path, _ = strings.Cut(s, "/")
		if u.Path != "" {
			u.Path = "/" + u.Path
		}
	}

	if strings.Contains(s, "@") {
		var auth string
		auth, s, _ = strings.Cut(s, "@")
		if user, pwd, ok := strings.Cut(auth, ":"); ok {
			u.User, _ = url.QueryUnescape(user)
			u.Password, _ = url.QueryUnescape(pwd)
		} else {
			u.User, _ = url.QueryUnescape(auth)
		}
	}

	//ipv6
	if strings.HasPrefix(s, "[") && strings.Contains(s, "]") {
		u.Host, u.Port, _ = strings.Cut(s, "]:")
		u.Host += "]"
	} else {
		u.Host, u.Port, _ = strings.Cut(s, ":")
	}

	if ue, e := url.PathUnescape(u.Path); e == nil {
		u.Path = ue
	}

	return
}

// RawURL
type RawURL struct {
	Scheme   string
	Host     string
	Port     string
	Path     string
	Query    string
	Fragment string
	User     string
	Password string
}

func (u RawURL) DirPath() string {
	return path.Dir(u.Path)
}

func (u RawURL) FileName() string {
	return path.Base(u.Path)
}

func (u RawURL) DirName() string {
	return path.Base(u.DirPath())
}

func (u RawURL) FileExt() string {
	return path.Ext(u.Path)
}

// 转换为字符串
func (u RawURL) String() string {
	var s bytes.Buffer

	if u.Scheme != "" {
		s.WriteString(u.Scheme)
		s.WriteString("://")
	}

	if u.Host != "" || u.Port != "" {
		if u.Scheme == "" {
			s.WriteString("//")
		}

		if u.User != "" {
			s.WriteString(u.User)
		}

		if u.Password != "" {
			s.WriteString(":")
			s.WriteString(u.Password)
		}

		if u.User != "" || u.Password != "" {
			s.WriteString("@")
		}

		if u.Host != "" {
			s.WriteString(u.Host)
		}

		if u.Port != "" {
			s.WriteString(":")
			s.WriteString(u.Port)
		}
	}

	if u.Path != "" {
		s.WriteString(u.Path)
	}

	if u.Query != "" {
		s.WriteString("?")
		s.WriteString(u.Query)
	}

	if u.Fragment != "" {
		s.WriteString("#")
		s.WriteString(u.Fragment)
	}

	return s.String()
}

// 合并路径，支持相对路径和绝对路径，如果原始路径不以"/"结尾（文件夹），则去掉文件部分再合并。
func (u RawURL) JoinPath(elem ...string) (r RawURL) {
	r = u
	if !strings.HasPrefix(elem[0], "/") {
		if !strings.HasSuffix(r.Path, "/") {
			r.Path = path.Dir(r.Path)
		}
		elem = append([]string{r.Path}, elem...)
	}
	r.Path = path.Join(elem...)
	return
}

// 移除查询参数
func (u RawURL) RemoveQuery() RawURL {
	u.Query = ""
	return u
}

// 移除片段
func (u RawURL) RemoveFragment() RawURL {
	u.Fragment = ""
	return u
}

// 更新
func (u RawURL) Set(options ...func(*RawURL)) (r RawURL) {
	r = u
	for _, option := range options {
		option(&r)
	}
	return
}

func Scheme(scheme string) func(u *RawURL) { return func(u *RawURL) { u.Scheme = scheme } }

func User(user string) func(u *RawURL) { return func(u *RawURL) { u.User = user } }

func Password(password string) func(u *RawURL) { return func(u *RawURL) { u.Password = password } }

func Host(host string) func(u *RawURL) { return func(u *RawURL) { u.Host = host } }

func Port(port string) func(u *RawURL) { return func(u *RawURL) { u.Port = port } }

func Path(path string) func(u *RawURL) { return func(u *RawURL) { u.Path = path } }

func Query(query string) func(u *RawURL) { return func(u *RawURL) { u.Query = query } }

func Fragment(fragment string) func(u *RawURL) { return func(u *RawURL) { u.Fragment = fragment } }
