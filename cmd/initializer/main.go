package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"wg-overlay/pkg/overlay"
	"wg-overlay/pkg/wireguard"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"k8s.io/klog/v2"
)

func main() {
	var (
		wireguardInterfaceName string
		listenPort             int
		nodeIP                 string
		overlayCIDR            string
		podCIDR                string
	)
	flag.StringVar(&wireguardInterfaceName, "wg-dev", "wg0", "the name of the wireguard interface on the host")
	flag.IntVar(&listenPort, "listen-port", 51820, "the listening port of the wireguard interface")
	flag.StringVar(&nodeIP, "node-ip", "", "IP of the kubernetes node")
	flag.StringVar(&overlayCIDR, "overlay-cidr", "100.64.0.0/16", "address space of the node overlay")
	flag.StringVar(&podCIDR, "pod-cidr", "10.244.0.0/16", "address space of the pods")
	flag.Parse()
	c, err := wgctrl.New()
	if err != nil {
		klog.Fatalf("failed to setup wgctrl client: %s", err)
	}

	dev, err := c.Device(wireguardInterfaceName)
	if err == nil {
		if dev.PublicKey.String() == "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" {
			klog.Warningf("%s found with empty public key, regenerating", wireguardInterfaceName)
		} else {
			klog.Infof("wireguard %s device configured with public key %s", dev.Name, dev.PublicKey.String())
			os.Exit(0)
		}
	}

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		klog.Fatalf("failed to check device %s", wireguardInterfaceName)
	}

	klog.Infof("wireguard device %s missing, setting up", wireguardInterfaceName)
	wgIP := overlay.OverlayIP(nodeIP, overlayCIDR)

	linkDev, err := setupWireGuardLinkDevice(wgIP, wireguardInterfaceName)
	if err != nil {
		klog.Fatalf("failed to setup new wireguard link device: %s", err)
	}

	cfg, err := newWireGuardConfig(wireguardInterfaceName, listenPort)
	if err != nil {
		klog.Fatalf("failed to create new wg config: %s", err)
	}

	if err = c.ConfigureDevice(wireguardInterfaceName, *cfg); err != nil {
		klog.Fatalf("failed to configure device %s: %s", wireguardInterfaceName, err)
	}

	if err = netlink.LinkSetUp(linkDev); err != nil {
		klog.Fatalf("failed to set link %s up: %s", wireguardInterfaceName, err)
	}

	if err = ensureRoutes(linkDev, overlayCIDR, podCIDR); err != nil {
		klog.Fatalf("failed to add routes: %s", err)
	}
	klog.Infof("%s dev configured", wireguardInterfaceName)
}

func ensureRoutes(link netlink.Link, routes ...string) error {
	for _, route := range routes {
		_, ipNet, err := net.ParseCIDR(route)
		if err != nil {
			return fmt.Errorf("failed to parse route %s: %w", route, err)
		}
		r := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       ipNet,
		}
		err = netlink.RouteAdd(&r)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("failed to add route %s: %w", route, err)
		}
	}
	return nil
}

func setupWireGuardLinkDevice(wireguardIP, ifName string) (netlink.Link, error) {
	la := netlink.LinkAttrs{
		TxQLen: -1, // default from NewAttrs()
		Name:   ifName,
	}
	wg := &wireguard.LinkWireGuard{LinkAttrs: la}

	err := netlink.LinkAdd(wg)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("failed to add link: %w", err)
	}

	ip := net.ParseIP(wireguardIP)
	addr := &netlink.Addr{
		IPNet: toIPNet(ip),
	}

	err = netlink.AddrAdd(wg, addr)
	if err != nil {
		return nil,
			fmt.Errorf("failed to add address %s to link dev %s: %w", addr.IP.String(), wg.LinkAttrs.Name, err)
	}
	return wg, nil
}

func newWireGuardConfig(wireguardInterfaceName string, port int) (*wgtypes.Config, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	return &wgtypes.Config{
		PrivateKey: &privateKey,
		ListenPort: &port,
	}, nil
}

func toIPNet(ip net.IP) *net.IPNet {
	ipNet := &net.IPNet{
		IP: ip,
	}
	if ip.To4() != nil {
		ipNet.Mask = net.CIDRMask(32, 32)
	} else {
		ipNet.Mask = net.CIDRMask(128, 128)
	}
	return ipNet
}
