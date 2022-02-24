package overlay

import "context"

type OverlayNetworkService interface {
	Run(ctx context.Context)
}
