package wireguard

import (
	v1 "k8s.io/api/core/v1"
)

const (
	WireguardPublicKeyAnnotationName = "wireguard-publickey"
	WireguardIPAnnotationName        = "wireguard-ip"
	DefaultListenPort                = 51820
)

type Host struct {
	Address    string
	PrivateKey string
	PublicKey  string
	ListenPort int
}

func NewHost(overlayip string) (Host, error) {
	return Host{Address: overlayip}, nil
}

func (hostInterface Host) Annotate(selfNode *v1.Node) (bool, error) {
	update := false
	if ip, ok := selfNode.Annotations[WireguardIPAnnotationName]; !ok || ip != hostInterface.Address {
		selfNode.Annotations[WireguardIPAnnotationName] = hostInterface.Address
		update = true
	}

	pubKey := hostInterface.PublicKey
	if pub, ok := selfNode.Annotations[WireguardPublicKeyAnnotationName]; !ok || pub != pubKey {
		selfNode.Annotations[WireguardPublicKeyAnnotationName] = pubKey
		update = true
	}
	return update, nil
}
