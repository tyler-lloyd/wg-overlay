package wireguard

import "github.com/vishvananda/netlink"

type LinkWireGuard struct {
	LinkAttrs netlink.LinkAttrs
}

func (w *LinkWireGuard) Attrs() *netlink.LinkAttrs {
	return &w.LinkAttrs
}
func (w *LinkWireGuard) Type() string {
	return "wireguard"
}
