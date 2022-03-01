package wireguard

import (
	"os"
	"strings"

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
	ListenPort int
}

func NewHost(overlayip string) (Host, error) {
	b, err := os.ReadFile(FileWireguardKeyPrivate)
	if err != nil {
		return Host{}, err
	}
	privateKey := string(b)
	privateKey = strings.TrimRight(privateKey, "\n")
	return Host{
		Address:    overlayip, //o.overlayConf.OverlayIP,
		PrivateKey: privateKey,
		ListenPort: DefaultListenPort,
	}, nil
}

func (hostInterface Host) Annotate(selfNode *v1.Node) (bool, error) {
	update := false
	if ip, ok := selfNode.Annotations[WireguardIPAnnotationName]; !ok || ip != hostInterface.Address {
		selfNode.Annotations[WireguardIPAnnotationName] = hostInterface.Address
		update = true
	}

	b, err := os.ReadFile(FileWireguardKeyPublic)
	if err != nil {
		return false, err
	}
	pubKey := string(b)
	pubKey = strings.TrimRight(pubKey, "\n")
	if pub, ok := selfNode.Annotations[WireguardPublicKeyAnnotationName]; !ok || pub != pubKey {
		selfNode.Annotations[WireguardPublicKeyAnnotationName] = pubKey
		update = true
	}
	return update, nil

}
