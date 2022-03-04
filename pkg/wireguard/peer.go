package wireguard

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	v1 "k8s.io/api/core/v1"
)

type Peer struct {
	PublicKey  string
	AllowedIPs []string
	Endpoint   string
}

func FromNode(node v1.Node) (*wgtypes.Peer, error) {
	wgIP, hasIP := node.Annotations[WireguardIPAnnotationName]
	if !hasIP {
		return nil, fmt.Errorf("%s missing %s", node.Name, WireguardIPAnnotationName)
	}
	publicKeyString, hasPubKey := node.Annotations[WireguardPublicKeyAnnotationName]
	if !hasPubKey {
		return nil, fmt.Errorf("%s missing %s", node.Name, WireguardPublicKeyAnnotationName)
	}
	publicKey, err := wgtypes.ParseKey(publicKeyString)
	if err != nil {
		return nil, fmt.Errorf("could not parse key %s", publicKeyString)
	}
	allowedCIDRs := node.Spec.PodCIDRs
	allowedCIDRs = append(allowedCIDRs, wgIP+"/32")

	allowedIPs := make([]net.IPNet, 0)

	for _, cidr := range allowedCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse cidr %s: %w", cidr, err)
		}
		allowedIPs = append(allowedIPs, *ipNet)
	}
	return &wgtypes.Peer{
		PublicKey:  publicKey,
		AllowedIPs: allowedIPs,
		Endpoint: &net.UDPAddr{
			IP:   net.ParseIP(getHostEndpoint(node)),
			Port: DefaultListenPort,
		},
	}, nil
}

func getHostEndpoint(node v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}
	panic("getHostEndpoint failed to get NodeInternalIP")
}
