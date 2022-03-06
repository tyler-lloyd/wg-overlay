package controllers

import (
	"context"
	"fmt"
	"wg-overlay/pkg/overlay"
	"wg-overlay/pkg/wireguard"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PeerOperation string

const (
	Add PeerOperation = "ADD"
	Del PeerOperation = "DEL"
)

type WireguardNodeReconciler struct {
	client.Client
	overlay.Config
	*wgtypes.Device
	WgClient *wgctrl.Client
	cache    map[string]wgtypes.Peer
	//Scheme *runtime.Scheme
}

func (r *WireguardNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var node v1.Node
	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		logger.Error(err, "unable to fetch node")
		// todo delete peer here if req.Name != r.NodeName but need a way to retrieve the public key
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if node.Name == r.NodeName {
		if update, err := r.Annotate(&node); update && err == nil {
			logger.Info("updating self annotations")
			r.Update(ctx, &node, &client.UpdateOptions{})
		} else if err != nil {
			logger.Error(err, "unable to annotate node")
		}
	} else if err := r.ReconcilePeer(node, Add); err != nil {
		logger.Error(err, "failed to reconcile peer") // todo probably ignore "not ready" nodes (i.e. missing pub key or ip)
	}

	return ctrl.Result{}, nil
}

func (r *WireguardNodeReconciler) Annotate(n *v1.Node) (bool, error) {
	update := false
	if ip, ok := n.Annotations[wireguard.IPAnnotationName]; !ok || ip != r.OverlayIP {
		n.Annotations[wireguard.IPAnnotationName] = r.OverlayIP
		update = true
	}

	pubKey := r.PublicKey.String()
	if pub, ok := n.Annotations[wireguard.PublicKeyAnnotationName]; !ok || pub != pubKey {
		n.Annotations[wireguard.PublicKeyAnnotationName] = pubKey
		update = true
	}
	return update, nil
}

func (r *WireguardNodeReconciler) ReconcilePeer(node v1.Node, op PeerOperation) error {
	peer, err := wireguard.FromNode(node)
	if err != nil {
		return fmt.Errorf("failed to get peer from node")
	}
	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  peer.PublicKey,
				AllowedIPs: peer.AllowedIPs,
				Endpoint:   peer.Endpoint,
			},
		},
	}
	switch op {
	case Add:
		if _, ok := r.cache[peer.PublicKey.String()]; ok {
			return nil
		}
	case Del:
		for i := range cfg.Peers {
			cfg.Peers[i].Remove = true
		}
	}
	err = r.WgClient.ConfigureDevice(r.Name, cfg)
	if err != nil {
		return fmt.Errorf("ConfigureDevice failed: %w", err)
	}
	if op == Add {
		r.cache[peer.PublicKey.String()] = *peer
	}
	if op == Del {
		delete(r.cache, peer.PublicKey.String())
	}
	return nil
}

func (r *WireguardNodeReconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

func (r *WireguardNodeReconciler) HydrateCache() {
	r.cache = make(map[string]wgtypes.Peer)
	for _, peer := range r.Peers {
		r.cache[peer.PublicKey.String()] = peer
	}
}
