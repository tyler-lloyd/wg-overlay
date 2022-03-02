package overlay

import (
	"wg-overlay/pkg/wireguard"

	"k8s.io/client-go/kubernetes"
)

func NewWireGuardNetworkService(config Config, k *kubernetes.Clientset) OverlayNetworkService {
	return &WireGuardNetworkService{
		overlayConf: config,
		cache:       map[string]wireguard.Peer{},
		kubeclient:  k,
	}
}
