package nx

import (
	"net"
	"strconv"
)

func ListenSplit(listen string) (host string, port int, err error) {
	var sport string
	if host, sport, err = net.SplitHostPort(listen); err != nil {
		return
	}

	if port, err = strconv.Atoi(sport); err != nil {
		return
	}

	if host == "" {
		host = "127.0.0.1"
	}

	return
}
