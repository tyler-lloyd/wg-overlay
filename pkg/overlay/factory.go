package overlay

func NewWireGuardNetworkService(config KubernetesConfig) OverlayNetworkService {
	return &WireGuardNetworkService{
		config:    config,
		nodeCache: map[string]string{},
	}
}
