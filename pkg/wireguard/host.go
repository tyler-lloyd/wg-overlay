package wireguard

const (
	PublicKeyAnnotationName = "wireguard-publickey"
	IPAnnotationName        = "wireguard-ip"
	DefaultListenPort       = 51820
)

// Host is a custom type for holding WireGuard state information about the host
type Host struct {
	Address    string
	PrivateKey string
	PublicKey  string
	ListenPort int
}

// NewHost returns a new host config.
func NewHost(overlayip string) (Host, error) {
	return Host{Address: overlayip}, nil
}
