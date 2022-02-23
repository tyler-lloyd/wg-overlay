package overlay

import (
	"context"
	"time"
)

const (
	syncInterval time.Duration = 5 * time.Second
)

type WireGuardNetworkService struct {
	config    KubernetesConfig
	nodeCache map[string]string
}

func (o *WireGuardNetworkService) Run(ctx context.Context) error {
	return nil
}

func (o *WireGuardNetworkService) initialize() {

}

func (o *WireGuardNetworkService) initializeCache() {

}
