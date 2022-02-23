package main

import (
	"aks-wireguard-overlay/pkg/overlay"
	"aks-wireguard-overlay/pkg/wireguard"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const WireguardConfigurationInterfaceTemplate = "[Interface]\nAddress = %s\nListenPort = %d\nPrivateKey = %s\n"
const WireguardConfigurationPeerTemplate = "[Peer]\nPublicKey = %s\nAllowedIPs = %s\nEndpoint = %s\n"

const (
	FileWireguardKeyPrivate = "/etc/wireguard/privatekey"
	FileWireguardKeyPublic  = "/etc/wireguard/publickey"
	FileWireguardWg0        = "/etc/wireguard/wg0.conf"
	FileWireguardUpdate     = "/etc/wireguard/update"
)

var addedPeers map[string]string

func main() {
	defer klog.Flush()
	addedPeers = make(map[string]string)
	nodeName := os.Getenv("NODE_NAME")
	nodeIP := os.Getenv("NODE_IP")
	cfg := overlay.KubernetesConfig{NodeName: nodeName, UnderlayIP: nodeIP, OverlayCIDR: "100.64.0.0/16"}
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	cfg.Client = client

	networkService := overlay.NewWireGuardNetworkService(cfg)
	err = networkService.Run()
	if err != nil {
		klog.Fatal(err)
	}
	klog.Info("wireguard network service shutting down")
	for {
		func() {
			klog.Info("starting sync...")
			defer time.Sleep(5 * time.Second)
			err := syncWireguardConfig()
			if err != nil {
				klog.Error(err)
				return
			}

			err = syncWireguardNode()
			if err != nil {
				klog.Error(err)
				return
			}

			err = syncWireguardPeers()
			if err != nil {
				klog.Error(err)
				return
			}
		}()
	}
}

func syncWireguardPeers() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	nodeName, err := os.ReadFile("/proc/sys/kernel/hostname")
	if err != nil {
		return err
	}
	self := string(nodeName)
	self = strings.TrimRight(self, "\n")
	nodes, err := client.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	var peers []wireguard.Peer
	for _, n := range nodes.Items {
		if n.Name == self {
			continue
		}

		ip, okIP := n.Annotations["wg-ip"]
		pub, okPub := n.Annotations["wg-pubkey"]

		if _, ok := addedPeers[pub]; okIP && okPub && !ok {
			peers = append(peers, wireguard.Peer{
				PublicKey:  pub,
				AllowedIPs: []string{fmt.Sprintf("%s/32", ip), n.Spec.PodCIDR},
				Endpoint:   n.Status.Addresses[1].Address + ":51820",
			})
			addedPeers[pub] = ip
		}
	}

	f, err := os.OpenFile(FileWireguardWg0, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, peer := range peers {
		klog.Infof("adding peer %q", peer)
		peerStr := fmt.Sprintf(WireguardConfigurationPeerTemplate, peer.PublicKey, strings.Join(peer.AllowedIPs, ", "), peer.Endpoint)
		f.WriteString(peerStr)
	}
	if len(peers) > 0 {
		err = os.WriteFile(FileWireguardUpdate, []byte("1"), 0644)
	}
	if err != nil {
		return err
	}
	return nil
}

func syncWireguardNode() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	nodeName, err := os.ReadFile("/proc/sys/kernel/hostname")
	if err != nil {
		return err
	}
	nodeNameStr := string(nodeName)
	nodeNameStr = strings.TrimRight(nodeNameStr, "\n")
	n, err := client.CoreV1().Nodes().Get(context.Background(), nodeNameStr, metav1.GetOptions{})
	if err != nil {
		return err
	}

	publicKey, _ := os.ReadFile(FileWireguardKeyPublic)
	key := string(publicKey)
	key = strings.TrimRight(key, "\n")
	nodeKey, ok := n.Annotations["wg-pubkey"]
	if !ok || nodeKey != key {
		n.Annotations["wg-pubkey"] = key
		_, err = client.CoreV1().Nodes().Update(context.Background(), n, metav1.UpdateOptions{})
	}
	if err != nil {
		return err
	}

	wgIP, err := getWireguardIP()
	if err != nil {
		return err
	}

	nodeKey, ok = n.Annotations["wg-ip"]
	if !ok || nodeKey != wgIP {
		n.Annotations["wg-ip"] = wgIP
		_, err = client.CoreV1().Nodes().Update(context.Background(), n, metav1.UpdateOptions{})
	}

	if err != nil {
		return err
	}
	return nil
}

func getWireguardIP() (string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	nodeName, err := os.ReadFile("/proc/sys/kernel/hostname")
	if err != nil {
		return "", err
	}
	nodeNameStr := string(nodeName)
	nodeNameStr = strings.TrimRight(nodeNameStr, "\n")
	n, err := client.CoreV1().Nodes().Get(context.Background(), nodeNameStr, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	var nodeIPv4Addr string
	for _, addr := range n.Status.Addresses {
		if strings.Count(addr.Address, ".") == 3 {
			klog.Infof("node IP %s", addr.Address)
			nodeIPv4Addr = addr.Address
			break
		}
	}
	addrs := strings.Split(nodeIPv4Addr, ".")
	addrs[0] = "100"
	addrs[1] = "64"
	nodeIPv4Addr = strings.Join(addrs, ".")
	return nodeIPv4Addr, nil
}

func syncWireguardConfig() error {
	_, err := os.ReadFile(FileWireguardWg0)
	if errors.Is(err, fs.ErrNotExist) {
		klog.Info("wg0.conf not found, creating...")
		pk, _ := os.ReadFile(FileWireguardKeyPrivate)
		wireguardIP, err := getWireguardIP()
		if err != nil {
			return err
		}
		configData := fmt.Sprintf(WireguardConfigurationInterfaceTemplate, wireguardIP, 51820, string(pk))

		err = os.WriteFile(FileWireguardWg0, []byte(configData), 0644)
		if err != nil {
			return err
		}
		klog.Info("successfully created wg0.conf")
		err = os.WriteFile(FileWireguardUpdate, []byte("1"), 0644)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}
