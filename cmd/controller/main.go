package main

import (
	"flag"
	"os"
	"wg-overlay/pkg/controllers"
	"wg-overlay/pkg/overlay"
	"wg-overlay/pkg/wireguard"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"golang.zx2c4.com/wireguard/wgctrl"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports
	v1 "k8s.io/api/core/v1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		publicKey            string
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&publicKey, "public-key", "", "The wireguard public key of the node.")

	config := overlay.Config{}
	flag.StringVar(&config.OverlayCIDR, "overlay-cidr", "100.64.0.0/16", "The wireguard overlay address space.")
	flag.StringVar(&config.NodeName, "node-name", "", "The node name this daemon is running on.")
	flag.StringVar(&config.UnderlayIP, "node-ip", "", "The ip of the node this daemon is running on.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         false,
	})
	utilruntime.Must(err)
	config.OverlayIP = overlay.OverlayIP(config.UnderlayIP, config.OverlayCIDR)
	hostInterface, err := wireguard.NewHost(config.OverlayIP)
	setupLog.Info("setup host", "interface", hostInterface, "overlaycfg", config)
	if err != nil {
		setupLog.Error(err, "unable to load wireguard host configuration")
		os.Exit(1)
	}

	wgClient, err := wgctrl.New()
	if err != nil {
		setupLog.Error(err, "unable to create wgCtrl client")
		os.Exit(1)
	}
	controller := &controllers.WireguardNodeReconciler{
		Host:     hostInterface,
		Config:   config,
		WgClient: wgClient,
	}
	err = builder.
		ControllerManagedBy(mgr).
		For(&v1.Node{}).
		Complete(controller)

	if err != nil {
		setupLog.Error(err, "failed to create the controller")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
