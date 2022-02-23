package wireguard

type WireguardConfiguration struct {
	HostInterface Host
	Peers []Peer
}