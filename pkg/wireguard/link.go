package wireguard

import "github.com/vishvananda/netlink"

// WireGuard link type
type LinkWireGuard struct {
	LinkAttrs netlink.LinkAttrs
}

// Attrs returns the attributes of the link
func (w *LinkWireGuard) Attrs() *netlink.LinkAttrs {
	return &w.LinkAttrs
}

// Type returns the link type
func (w *LinkWireGuard) Type() string {
	return "wireguard"
}
