package util

import (
	"net"
	"strconv"
)

func IsIP(str string) bool {
	return net.ParseIP(str) != nil
}

func IsPort(str string) bool {
	if i, err := strconv.Atoi(str); err == nil && i > 0 && i < 65536 {
		return true
	}
	return false
}
