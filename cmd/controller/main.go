package main

import (
	"flag"
	"os"
	"wg-overlay/pkg/controllers"
	"wg-overlay/pkg/overlay"

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
	corev1 "k8s.io/api/core/v1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		wgDeviceName         string
	)
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&wgDeviceName, "wg-dev", "wg0", "The device name of the wireguard interface.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

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

	c, err := wgctrl.New()
	if err != nil {
		setupLog.Error(err, "unable to create wgCtrl client")
		os.Exit(1)
	}
	wgDevice, err := c.Device(wgDeviceName)
	if err != nil {
		setupLog.Error(err, "unable to get wireguard device %s", wgDeviceName)
		os.Exit(1)
	}
	controller := &controllers.WireguardNodeReconciler{
		Device:   wgDevice,
		Config:   config,
		WgClient: c,
	}
	controller.HydrateCache()
	err = builder.
		ControllerManagedBy(mgr).
		For(&corev1.Node{}).
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
