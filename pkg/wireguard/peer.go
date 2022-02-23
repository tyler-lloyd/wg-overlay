package wireguard

type Peer struct {
	PublicKey  string
	AllowedIPs []string
	Endpoint   string
}
