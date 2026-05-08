package httpx

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

type StatusError struct {
	Status int
	Raw    []byte
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

	if se, ok := errors.AsType[*StatusError](err); ok {
		return se.Status == http.StatusRequestTimeout || se.Status == http.StatusTooManyRequests || se.Status >= 500
	}

	if _, ok := errors.AsType[net.Error](err); ok {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "temporary") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "EOF")
}
