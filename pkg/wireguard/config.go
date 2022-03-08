package wireguard

// Config is the WireGuard config on the host
type Config struct {
	HostInterface Host
	Peers         []Peer
}
