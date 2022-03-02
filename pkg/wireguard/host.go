package wireguard

import (
	"bufio"
	"fmt"
	"os"

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
	privateKey, err := getPrivateKey()
	if err != nil {
		return Host{}, err
	}

	publicKey, err := getPublicKey()
	if err != nil {
		return Host{}, err
	}
	return Host{
		Address:    overlayip, //o.overlayConf.OverlayIP,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		ListenPort: DefaultListenPort,
	}, nil
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

func getPublicKey() (string, error) {
	return getKeyFile(FileWireguardKeyPublic)
}

func getPrivateKey() (string, error) {
	return getKeyFile(FileWireguardKeyPrivate)
}

func getKeyFile(filename string) (string, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	scanner := bufio.NewScanner(fd)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", fmt.Errorf("failed to scan first line of file %s", filename)
}
