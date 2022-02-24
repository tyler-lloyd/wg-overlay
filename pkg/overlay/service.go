package overlay

import (
	"aks-wireguard-overlay/pkg/wireguard"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	syncInterval                            = 5 * time.Second
	DefaultListenPort                       = 51820
	WireguardConfigurationInterfaceTemplate = "[Interface]\nAddress = %s\nListenPort = %d\nPrivateKey = %s\n"
	WireguardConfigurationPeerTemplate      = "[Peer]\nPublicKey = %s\nAllowedIPs = %s\nEndpoint = %s\n"
)

type WireGuardNetworkService struct {
	overlayConf   Config
	wireguardConf wireguard.Config
	cache         map[string]wireguard.Peer
	kubeclient    *kubernetes.Clientset
}

func (o *WireGuardNetworkService) Run(ctx context.Context) {
	o.initialize()
	for {
		func() {
			defer time.Sleep(syncInterval)
			err := o.syncHost()
			if err != nil {
				klog.Errorf("syncHost failure: %s", err)
			}
			err = o.syncPeers()
			if err != nil {
				klog.Errorf("syncPeers failed: %s", err)
				return
			}
			err = o.syncWireguardConfig()
			if err != nil {
				klog.Errorf("syncWireguardConfig failure: %s", err)
				return
			}
		}()
	}
}

func (o *WireGuardNetworkService) initialize() {
	o.syncCache()
}

func (o *WireGuardNetworkService) syncCache() {
	c, err := wireguard.ParseConfFile(wireguard.DefaultWireGuardConf)
	if err != nil {
		klog.Errorf("failed to parse wireguard conf %s: %s", wireguard.DefaultWireGuardConf, err)
		return
	}
	o.wireguardConf = c
	for _, peer := range c.Peers {
		o.cache[peer.PublicKey] = peer
	}
}

func (o *WireGuardNetworkService) syncHost() error {
	pk, err := os.ReadFile(wireguard.FileWireguardKeyPrivate)
	if err != nil {
		return err
	}
	hostInterface := wireguard.Host{
		Address:    o.overlayConf.OverlayIP,
		PrivateKey: string(pk),
		ListenPort: DefaultListenPort,
	}
	o.wireguardConf.HostInterface = hostInterface

	selfNode, err := o.kubeclient.CoreV1().Nodes().Get(context.Background(), o.overlayConf.NodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if ip, ok := selfNode.Annotations[wireguard.WireguardIPAnnotationName]; !ok || ip != hostInterface.Address {
		selfNode.Annotations[wireguard.WireguardIPAnnotationName] = hostInterface.Address
	}

	pubKey, err := os.ReadFile(wireguard.FileWireguardKeyPublic)
	if err != nil {
		return err
	}
	if pub, ok := selfNode.Annotations[wireguard.WireguardPublicKeyAnnotationName]; !ok || pub != string(pubKey) {
		selfNode.Annotations[wireguard.WireguardPublicKeyAnnotationName] = string(pubKey)
	}

	_, err = o.kubeclient.CoreV1().Nodes().Update(context.Background(), selfNode, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (o *WireGuardNetworkService) syncPeers() error {
	nodes, err := o.kubeclient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	o.wireguardConf.Peers = []wireguard.Peer{}

	for _, node := range nodes.Items {
		if node.Name == o.overlayConf.NodeName {
			continue
		}

		wgIP, hasIP := node.Annotations[wireguard.WireguardIPAnnotationName]
		publicKey, hasPubKey := node.Annotations[wireguard.WireguardPublicKeyAnnotationName]
		if hasIP && hasPubKey {
			peer := wireguard.Peer{
				PublicKey:  publicKey,
				AllowedIPs: []string{wgIP + "/32", node.Spec.PodCIDR},
				Endpoint:   getHostEndpoint(node),
			}
			o.wireguardConf.Peers = append(o.wireguardConf.Peers, peer)
		}
	}
	return nil
}

func getHostEndpoint(node corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func (o *WireGuardNetworkService) syncWireguardConfig() error {
	actual, err := wireguard.ParseConfFile(wireguard.DefaultWireGuardConf)
	if err != nil {
		return err
	}
	needWgQuickRestart := false
	if actual.HostInterface != o.wireguardConf.HostInterface {
		klog.Infof("actual interface %q does not match goal interface %q", actual.HostInterface, o.wireguardConf.HostInterface)
		needWgQuickRestart = true
	}

	actualPeers := make(map[string]bool)
	for _, peer := range actual.Peers {
		actualPeers[peer.PublicKey] = true // todo should hash the whole struct as key
	}

	for _, peer := range o.wireguardConf.Peers {
		if _, ok := actualPeers[peer.PublicKey]; !ok {
			needWgQuickRestart = true
		}
	}

	if needWgQuickRestart {
		s := strings.Builder{}

		s.WriteString(fmt.Sprintf(WireguardConfigurationInterfaceTemplate, o.wireguardConf.HostInterface.Address, DefaultListenPort, o.wireguardConf.HostInterface.PrivateKey))

		for _, peer := range o.wireguardConf.Peers {
			s.WriteString(fmt.Sprintf(WireguardConfigurationPeerTemplate, peer.PublicKey, strings.Join(peer.AllowedIPs, ","), peer.Endpoint))
		}

		err = os.WriteFile(wireguard.DefaultWireGuardConf, []byte(s.String()), 0644)
		if err != nil {
			return err
		}
		err = os.WriteFile(wireguard.FileWireguardUpdate, []byte("1"), 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
