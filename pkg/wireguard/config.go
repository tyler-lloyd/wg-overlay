package wireguard

type Config struct {
	HostInterface Host
	Peers         []Peer
}
