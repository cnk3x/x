package httpx

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type StatusError struct {
	Status     int
	Raw        []byte
	RetryAfter time.Duration // 429 响应中的 Retry-After 头
}

func (se *StatusError) Error() string {
	switch se.Status {
	case http.StatusUnauthorized:
		return fmt.Sprintf("需要登录: %d", se.Status)
	case http.StatusForbidden:
		return fmt.Sprintf("权限不足: %d", se.Status)
	case http.StatusNotFound:
		return fmt.Sprintf("地址错误: %d", se.Status)
	case http.StatusMethodNotAllowed:
		return fmt.Sprintf("方法错误: %d", se.Status)
	case http.StatusRequestTimeout:
		return fmt.Sprintf("请求超时: %d", se.Status)
	case http.StatusTooManyRequests:
		return fmt.Sprintf("请求太快: %d", se.Status)
	case http.StatusInternalServerError:
		return fmt.Sprintf("服务错误: %d", se.Status)
	case http.StatusBadGateway:
		return fmt.Sprintf("网关错误: %d", se.Status)
	case http.StatusServiceUnavailable:
		return fmt.Sprintf("服务错误: %d", se.Status)
	case http.StatusGatewayTimeout:
		return fmt.Sprintf("网关错误: %d", se.Status)
	default:
		return fmt.Sprintf("错误状态: %d", se.Status)
	}
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}

	var se *StatusError
	if errors.As(err, &se) {
		return se.Status == http.StatusRequestTimeout ||
			se.Status == http.StatusTooManyRequests ||
			se.Status == 425 || // Too Early
			se.Status >= 500
	}

	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "temporary") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "EOF")
}

// parseRetryAfter 解析 429 响应中的 Retry-After 头
// 返回应该等待的时长，如果无法解析则返回 0
func parseRetryAfter(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	// 尝试解析为秒数
	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	// 尝试解析为 HTTP 日期格式
	if t, err := http.ParseTime(retryAfter); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 0
}
