package controllers

import (
	"context"
	"wg-overlay/pkg/overlay"
	"wg-overlay/pkg/wireguard"

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WireguardNodeReconciler struct {
	client.Client
	WgHost      wireguard.Host
	OverlayConf overlay.Config
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

	update, err := r.WgHost.Annotate(&node)
	if err != nil {
		logger.Error(err, "unable to annotate node")
	}
	if update && node.Name == r.OverlayConf.NodeName {
		logger.Info("updating self annotations")
		r.Update(ctx, &node, &client.UpdateOptions{})
	}
	return ctrl.Result{}, nil
}

func (r *WireguardNodeReconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}
