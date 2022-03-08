package overlay

// Config is the overlay config for a given node
type Config struct {
	OverlayCIDR string
	OverlayIP   string
	UnderlayIP  string
	NodeName    string
}
