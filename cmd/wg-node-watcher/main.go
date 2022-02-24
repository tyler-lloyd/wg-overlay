package main

import (
	"aks-wireguard-overlay/pkg/overlay"
	"context"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const (
	FileWireguardKeyPrivate = "/etc/wireguard/privatekey"
	FileWireguardKeyPublic  = "/etc/wireguard/publickey"
	FileWireguardWg0        = "/etc/wireguard/wg0.conf"
	FileWireguardUpdate     = "/etc/wireguard/update"
)

func main() {
	defer klog.Flush()
	nodeName := os.Getenv("NODE_NAME")
	nodeIP := os.Getenv("NODE_IP")
	overlayCIDR := os.Getenv("OVERLAY_CIDR")
	cfg := overlay.Config{
		NodeName:    nodeName,
		UnderlayIP:  nodeIP,
		OverlayCIDR: overlayCIDR,
		OverlayIP:   overlay.OverlayIP(nodeIP, overlayCIDR),
	}

	kubeClient := newKubeClient()
	ctx := context.Background()
	wgs := overlay.NewWireGuardNetworkService(cfg, kubeClient)
	wgs.Run(ctx)
}

func newKubeClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatal(err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}
	return client
}
