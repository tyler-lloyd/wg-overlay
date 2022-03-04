package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"wg-overlay/pkg/overlay"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"k8s.io/klog/v2"
)

type WireGuard struct {
	LinkAttrs netlink.LinkAttrs
}

func (w *WireGuard) Attrs() *netlink.LinkAttrs {
	return &w.LinkAttrs
}
func (w *WireGuard) Type() string {
	return "wireguard"
}

func main() {
	var (
		wireguardInterfaceName string
		listenPort             int
		nodeIP                 string
		overlayCIDR            string
	)
	flag.StringVar(&wireguardInterfaceName, "wg-dev", "wg0", "the name of the wireguard interface on the host")
	flag.IntVar(&listenPort, "listen-port", 51820, "the listening port of the wireguard interface")
	flag.StringVar(&nodeIP, "node-ip", "", "IP of the kubernetes node")
	flag.StringVar(&overlayCIDR, "overlay-cidr", "100.64.0.0/16", "address space of the node overlay")
	flag.Parse()
	c, err := wgctrl.New()
	if err != nil {
		klog.Fatalf("failed to setup wgctrl client: %s", err)
	}

	dev, err := c.Device(wireguardInterfaceName)
	if err == nil {
		klog.Infof("wireguard %s device configured with public key %s", dev.Name, dev.PublicKey.String())
		os.Exit(0)
	}

	if !errors.Is(err, os.ErrNotExist) {
		klog.Fatalf("failed to check device %s", wireguardInterfaceName)
	}

	klog.Infof("%s device missing, setting up device", wireguardInterfaceName)
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
	klog.Infof("%s dev configured", wireguardInterfaceName)
}

func setupWireGuardLinkDevice(wireguardIP, ifName string) (netlink.Link, error) {
	la := netlink.NewLinkAttrs()
	la.Name = ifName
	wg := &WireGuard{LinkAttrs: la}
	err := netlink.LinkAdd(wg)
	if err != nil {
		return nil, fmt.Errorf("failed to add link: %w", err)
	}
	_, ipNet, err := net.ParseCIDR(wireguardIP + "/32")
	if err != nil {
		return nil, fmt.Errorf("failed to parse cidr: %w", err)
	}
	netlink.AddrAdd(wg, &netlink.Addr{
		IPNet: ipNet,
	})
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
