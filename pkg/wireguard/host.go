package wireguard

const (
	PublicKeyAnnotationName = "wireguard-publickey"
	IPAnnotationName        = "wireguard-ip"
	DefaultListenPort       = 51820
)

type Host struct {
	Address    string
	PrivateKey string
	PublicKey  string
	ListenPort int
}

func NewHost(overlayip string) (Host, error) {
	return Host{Address: overlayip}, nil
}
