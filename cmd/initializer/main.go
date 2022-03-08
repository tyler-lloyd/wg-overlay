package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/tyler-lloyd/wg-overlay/pkg/overlay"
	"github.com/tyler-lloyd/wg-overlay/pkg/wireguard"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"k8s.io/klog/v2"
)

// EmptyWireGuardKey is the empty, 256 bit WireGuard key
const EmptyWireGuardKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

// ensureRoutes takes destinations in the form of IP or CIDR and will
// add them as destination routes on the provided link device.
func ensureRoutes(link netlink.Link, destinations ...string) error {
	for _, dst := range destinations {
		klog.Infof("ensuring route for %s", dst)
		var dstIPNet *net.IPNet
		if ip := net.ParseIP(dst); ip != nil {
			dstIPNet = toIPNet(ip)
		}

		if _, ipNet, err := net.ParseCIDR(dst); err == nil {
			dstIPNet = ipNet
		}

		err := ensureRoute(link, dstIPNet)
		if err != nil {
			return fmt.Errorf("ensureRoute failed for %s: %w", dst, err)
		}
	}
	return nil
}

// ensureRoute will attempt to add the destination to the link device if it doesn't exist.
// if the route already exists then this is a no-op. An error is returned for all other errors.
func ensureRoute(link netlink.Link, destination *net.IPNet) error {
	r := netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       destination,
	}

	err := netlink.RouteAdd(&r)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("failed to add route %q: %w", destination, err)
	}
	return nil
}

// setupWireGuardLinkDevice ensures the device has an interface of type `wireguard`
// and adds the wireguardIP to the device. On Linux this is equivalent to
// `ip link add dev $ifName type wireguard && ip address add dev $ifName $wireguardIP`.
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

// newWireGuardConfig generates a new WireGuard host configuration with a private key and port
func newWireGuardConfig(port int) (*wgtypes.Config, error) {
	privateKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	return &wgtypes.Config{
		PrivateKey: &privateKey,
		ListenPort: &port,
	}, nil
}

// toIPNet converts an IP to a net.IPNet (CIDR) with the longest possible prefix length
// depending on the IP version, /32 for IPv4 and /128 for IPv6
func toIPNet(ip net.IP) *net.IPNet {
	ipNet := &net.IPNet{
		IP: ip,
	}
	if ip.To4() != nil {
		ipNet.Mask = net.CIDRMask(8*net.IPv4len, 8*net.IPv4len)
	} else {
		ipNet.Mask = net.CIDRMask(8*net.IPv6len, 8*net.IPv6len)
	}
	return ipNet
}

func main() {
	var (
		wgDeviceName string
		listenPort   int
		nodeIP       string
		overlayCIDR  string
		podCIDR      string
	)
	flag.StringVar(&wgDeviceName, "wg-dev", "wg0", "the name of the WireGuard interface on the host")
	flag.IntVar(&listenPort, "listen-port", 51820, "the listening port of the WireGuard interface")
	flag.StringVar(&nodeIP, "node-ip", "", "IP of the kubernetes node")
	flag.StringVar(&overlayCIDR, "overlay-cidr", "100.64.0.0/16", "address space of the node overlay")
	flag.StringVar(&podCIDR, "pod-cidr", "10.244.0.0/16", "address space of the pods")
	flag.Parse()

	c, err := wgctrl.New()
	if err != nil {
		klog.Fatalf("failed to setup wgctrl client: %s", err)
	}

	dev, err := c.Device(wgDeviceName)
	if err == nil {
		if dev.PublicKey.String() == EmptyWireGuardKey {
			klog.Warningf("%s found with empty public key, regenerating", wgDeviceName)
		} else {
			klog.Infof("WireGuard %s device configured with public key %s", dev.Name, dev.PublicKey.String())
			os.Exit(0)
		}
	}

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		klog.Fatalf("failed to check device %s", wgDeviceName)
	}

	klog.Infof("WireGuard device %s missing, setting up", wgDeviceName)
	wgIP := overlay.OverlayIP(nodeIP, overlayCIDR)

	linkDev, err := setupWireGuardLinkDevice(wgIP, wgDeviceName)
	if err != nil {
		klog.Fatalf("failed to setup new WireGuard link device: %s", err)
	}

	cfg, err := newWireGuardConfig(listenPort)
	if err != nil {
		klog.Fatalf("failed to create new wg config: %s", err)
	}

	if err = c.ConfigureDevice(wgDeviceName, *cfg); err != nil {
		klog.Fatalf("failed to configure device %s: %s", wgDeviceName, err)
	}

	if err = netlink.LinkSetUp(linkDev); err != nil {
		klog.Fatalf("failed to set link %s up: %s", wgDeviceName, err)
	}

	if err = ensureRoutes(linkDev, overlayCIDR, podCIDR); err != nil {
		klog.Fatalf("failed to add routes: %s", err)
	}
	klog.Infof("%s dev configured", wgDeviceName)
}
