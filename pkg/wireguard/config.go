package wireguard

const (
	FileWireguardKeyPrivate = "/etc/wireguard/privatekey"
	FileWireguardKeyPublic  = "/etc/wireguard/publickey"
	DefaultWireGuardConf    = "/etc/wireguard/wg0.conf"
	FileWireguardUpdate     = "/etc/wireguard/update"
)

type WireguardConfiguration struct {
	HostInterface Host
	Peers         []Peer
}
