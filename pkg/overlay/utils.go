package overlay

import (
	"net"
)

// OverlayIP generates an IP to be used in an overlay network
func OverlayIP(hostIP, overlayCIDR string) string {
	// todo implment proper ip allocation
	_, overlayNet, err := net.ParseCIDR(overlayCIDR)
	if err != nil {
		return ""
	}

	if overlayNet.IP.To4() == nil {
		// implement rfc 4193 ipv6 generation
		return ""
	}

	overlayIP := overlayNet.IP.To4()
	ip := net.ParseIP(hostIP).To4()
	ip[0] = overlayIP[0]
	ip[1] = overlayIP[1]
	return ip.String()
}
