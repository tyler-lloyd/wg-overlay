package wireguard

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	v1 "k8s.io/api/core/v1"
)

// Peer is a custom WireGuard peer type.
// todo(tyler-lloyd) remove usage eventually and only use wgctrl/wgtypes.Peer
type Peer struct {
	PublicKey  string
	AllowedIPs []string
	Endpoint   string
}

// FromNode returns a valid WireGuard peer if the node has the necessary metadata
func FromNode(node v1.Node) (*wgtypes.Peer, error) {
	publicKeyString, wgIP, err := publicKeyAndEndpoint(node)
	if err != nil {
		return nil, fmt.Errorf("failed to extract public key and endpoint: %w", err)
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

func publicKeyAndEndpoint(node v1.Node) (string, string, error) {
	ip, okIP := node.Annotations[IPAnnotationName]
	if !okIP {
		return "", "", fmt.Errorf("node %s missing %s", node.Name, IPAnnotationName)
	}

	pubKey, okPubKey := node.Annotations[PublicKeyAnnotationName]
	if !okPubKey {
		return "", "", fmt.Errorf("node %s missing %s", node.Name, PublicKeyAnnotationName)
	}
	return pubKey, ip, nil
}

func getHostEndpoint(node v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}
	panic("getHostEndpoint failed to get NodeInternalIP")
}
