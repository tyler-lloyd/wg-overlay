package overlay

import "k8s.io/client-go/kubernetes"

type KubernetesConfig struct {
	OverlayCIDR string
	UnderlayIP  string
	NodeName    string
	Client      *kubernetes.Clientset
}
