package overlay

import (
	"strings"
)

func OverlayIP(hostIP, overlayCIDR string) string {
	// todo implment proper ip allocation
	if strings.Count(hostIP, ":") > 2 {
		return "fd11:ffff:aaaa::1"
	}
	ip := strings.Split(hostIP, ".")
	ip[0] = "100"
	ip[1] = "64"
	return strings.Join(ip, ".")
}
