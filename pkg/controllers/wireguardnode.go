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
	wireguard.Host
	overlay.Config
	WgClient *wgctrl.Client
	cache    map[string]wgtypes.Peer
	//Scheme *runtime.Scheme
}

func (r *WireguardNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var node v1.Node
	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		logger.Error(err, "unable to fetch node")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if node.Name == r.Config.NodeName {
		if update, err := r.Host.Annotate(&node); update && err == nil {
			logger.Info("updating self annotations")
			r.Update(ctx, &node, &client.UpdateOptions{})
		} else if err != nil {
			logger.Error(err, "unable to annotate node")
		}
	} else {
		peer, err := wireguard.FromNode(node)
		if err != nil {
			logger.Error(err, "unable to get peer from node")
		} else {
			r.ReconcilePeer(*peer, Add)
		}
	}

	return ctrl.Result{}, nil
}

func (r *WireguardNodeReconciler) ReconcilePeer(peer wgtypes.Peer, op PeerOperation) error {
	cfg := wgtypes.Config{}
	if op == Del {
		cfg.Peers = []wgtypes.PeerConfig{
			{
				PublicKey: peer.PublicKey,
				Remove:    true,
			},
		}
	}
	if op == Add {
		if _, ok := r.cache[peer.PublicKey.String()]; ok {
			return nil
		}
		cfg.Peers = []wgtypes.PeerConfig{
			{
				PublicKey:  peer.PublicKey,
				AllowedIPs: peer.AllowedIPs,
				Endpoint:   peer.Endpoint,
			},
		}
	}
	err := r.WgClient.ConfigureDevice("wg0", cfg)
	if err != nil {
		return fmt.Errorf("ConfigureDevice failed: %w", err)
	}
	if op == Add {
		r.cache[peer.PublicKey.String()] = peer
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

func (r *WireguardNodeReconciler) HydrateCache(ctx context.Context) {
	dev, err := r.WgClient.Device("wg0")
	r.cache = make(map[string]wgtypes.Peer)
	if err != nil {
		log.FromContext(ctx).Error(err, "failed to get device", "dev", "wg0")
		return
	}

	for _, peer := range dev.Peers {
		
		r.cache[peer.PublicKey.String()] = peer
	}
}
