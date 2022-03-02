package overlay

import (
	"net"
	"strings"
)

func OverlayIP(hostIP, overlayCIDR string) string {
	// todo implment proper ip allocation
	if strings.Count(hostIP, ":") > 2 {
		// implement rfc 4193 ipv6 generation
		return ""
	}
	_, overlayNet, err := net.ParseCIDR(overlayCIDR)
	if err != nil {
		return ""
	}

	overlayIP := overlayNet.IP.To4()
	ip := net.ParseIP(hostIP).To4()
	ip[0] = overlayIP[0]
	ip[1] = overlayIP[1]
	return ip.String()
}