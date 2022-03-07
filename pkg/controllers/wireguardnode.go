package controllers

import (
	"context"
	"fmt"
	"sync"
	"wg-overlay/pkg/overlay"
	"wg-overlay/pkg/wireguard"

	"k8s.io/apimachinery/pkg/api/errors"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WireguardNodeReconciler struct {
	client.Client
	overlay.Config
	WgDevice    *wgtypes.Device
	WgClient    *wgctrl.Client
	cache       sync.Map
	mu          sync.Mutex
	initialized bool
}

func (r *WireguardNodeReconciler) syncCache(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.initialized {
		r.hydrateCache(ctx)
		r.initialized = true
	}
}

func (r *WireguardNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	r.syncCache(ctx)
	var node v1.Node
	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "unable to fetch node")
			return ctrl.Result{}, err
		}
		if pubKey, ok := r.cache.Load(req.Name); ok {
			key, err := wgtypes.ParseKey(pubKey.(string))
			if err != nil {
				logger.Error(err, "failed to parse public key", "key", pubKey)
				return ctrl.Result{}, err
			}
			peerToDelete := &wgtypes.Peer{PublicKey: key}

			err = r.ReconcilePeer(peerToDelete, true)
			if err != nil {
				logger.Error(err, "failed to delete peer")
				return ctrl.Result{}, err
			}
			r.cache.Delete(req.Name)
		}
		return ctrl.Result{}, nil
	}

	if node.Name == r.NodeName {
		if update, err := r.Annotate(&node); update && err == nil {
			err = r.Update(ctx, &node, &client.UpdateOptions{})
			if err != nil {
				logger.Error(err, "failed to update annotations on host node")
				return ctrl.Result{Requeue: true}, err
			}
		} else if err != nil {
			logger.Error(err, "unable to annotate node")
		}
	} else {
		peer, err := wireguard.FromNode(node)
		if err != nil {
			logger.Error(err, "failed to get peer from node")
			return ctrl.Result{}, nil
		}
		if pubKey, ok := r.cache.Load(node.Name); ok && pubKey.(string) == peer.PublicKey.String() {
			logger.Info("node already configured as peer", "publickey", peer.PublicKey.String())
			return ctrl.Result{}, nil
		}
		err = r.ReconcilePeer(peer, false)
		if err != nil {
			logger.Error(err, "failed to reconcile peer")
			return ctrl.Result{}, err
		}
		logger.Info("successfully added peer", "peer", *peer)
		r.cache.Store(node.Name, peer.PublicKey.String())
	}

	return ctrl.Result{}, nil
}

func (r *WireguardNodeReconciler) Annotate(n *v1.Node) (bool, error) {
	update := false
	if ip, ok := n.Annotations[wireguard.IPAnnotationName]; !ok || ip != r.OverlayIP {
		n.Annotations[wireguard.IPAnnotationName] = r.OverlayIP
		update = true
	}

	pubKey := r.WgDevice.PublicKey.String()
	if pub, ok := n.Annotations[wireguard.PublicKeyAnnotationName]; !ok || pub != pubKey {
		n.Annotations[wireguard.PublicKeyAnnotationName] = pubKey
		update = true
	}
	return update, nil
}

func (r *WireguardNodeReconciler) ReconcilePeer(peer *wgtypes.Peer, isDelete bool) error {
	cfg := wgtypes.Config{
		Peers: []wgtypes.PeerConfig{
			{
				PublicKey:  peer.PublicKey,
				AllowedIPs: peer.AllowedIPs,
				Endpoint:   peer.Endpoint,
			},
		},
	}
	if isDelete {
		for i := range cfg.Peers {
			cfg.Peers[i].Remove = true
		}
	}
	err := r.WgClient.ConfigureDevice(r.WgDevice.Name, cfg)
	if err != nil {
		return fmt.Errorf("ConfigureDevice failed: %w", err)
	}
	return nil
}

func (r *WireguardNodeReconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

func (r *WireguardNodeReconciler) hydrateCache(ctx context.Context) {
	if r.Client == nil {
		log.FromContext(ctx).Info("client not initialized, cannot load cache")
		return
	}
	var nodes v1.NodeList
	err := r.Client.List(ctx, &nodes)
	if err != nil {
		log.FromContext(ctx).Error(err, "could not list nodes")
		return
	}

	knownPeers := make(map[string]bool)
	for _, peer := range r.WgDevice.Peers {
		knownPeers[peer.PublicKey.String()] = true
	}

	r.cache = sync.Map{}
	for _, n := range nodes.Items {
		publicKey := n.Annotations[wireguard.PublicKeyAnnotationName]
		if ok := knownPeers[publicKey]; ok && publicKey != "" {
			r.cache.Store(n.Name, publicKey)
		}
	}
}
