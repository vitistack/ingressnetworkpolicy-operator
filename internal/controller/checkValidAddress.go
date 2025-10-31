package controller

import (
	"net"
	"strings"
)

func checkValidCIDR(ip string) bool {
	_, _, parsedCIDR := net.ParseCIDR(strings.TrimSpace(ip))
	if parsedCIDR == nil {
		return true
	} else {
		return false
	}
}
