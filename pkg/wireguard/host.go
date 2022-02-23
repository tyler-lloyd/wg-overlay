package wireguard

const (
	WireguardPublicKeyAnnotationName = "wireguard-publickey"
	WireguardIPAnnotationName        = "wireguard-ip"
)

type Host struct {
	Address    string
	PrivateKey string
	ListenPort int
	SaveConfig *bool
}
