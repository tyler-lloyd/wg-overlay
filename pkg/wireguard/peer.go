package wireguard

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

type Peer struct {
	PublicKey  string
	AllowedIPs []string
	Endpoint   string
}

func FromNode(node v1.Node) (*Peer, error) {
	wgIP, hasIP := node.Annotations[WireguardIPAnnotationName]
	if !hasIP {
		return nil, fmt.Errorf("%s missing %s", node.Name, WireguardIPAnnotationName)
	}
	publicKey, hasPubKey := node.Annotations[WireguardPublicKeyAnnotationName]
	if !hasPubKey {
		return nil, fmt.Errorf("%s missing %s", node.Name, WireguardPublicKeyAnnotationName)
	}
	allowedIPs := node.Spec.PodCIDRs
	allowedIPs = append(allowedIPs, wgIP+"/32")
	return &Peer{
		PublicKey:  publicKey,
		AllowedIPs: allowedIPs,
		Endpoint:   fmt.Sprintf("%s:%d", getHostEndpoint(node), DefaultListenPort),
	}, nil
}

func getHostEndpoint(node v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}
	return "" //error?
}
