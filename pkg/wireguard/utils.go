package wireguard

import (
	"fmt"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
)

func GetConfig(deviceName string) (*Config, error) {
	link, err := netlink.LinkByName(deviceName)
	if err != nil {
		return nil, err
	}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("link %s has no addresses", link.Attrs().Name)
	}

	wgIP := addrs[0].IP.String()

	c, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	dev, err := c.Device(deviceName)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		HostInterface: Host{
			Address:    wgIP,
			PrivateKey: dev.PrivateKey.String(),
			PublicKey:  dev.PublicKey.String(),
			ListenPort: dev.ListenPort,
		},
	}

	cfg.Peers = make([]Peer, 0)
	for _, peer := range dev.Peers {
		knownPeer := Peer{
			PublicKey: peer.PublicKey.String(),
			Endpoint:  peer.Endpoint.String(),
		}

		knownPeer.AllowedIPs = make([]string, 0)
		for _, ipNet := range peer.AllowedIPs {
			knownPeer.AllowedIPs = append(knownPeer.AllowedIPs, ipNet.String())
		}

		cfg.Peers = append(cfg.Peers, knownPeer)
	}

	return cfg, nil
}
