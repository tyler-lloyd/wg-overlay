package wireguard

const (
	FileWireguardKeyPrivate = "/etc/wireguard/privatekey"
	FileWireguardKeyPublic  = "/etc/wireguard/publickey"
	DefaultWireGuardConf    = "/etc/wireguard/wg0.conf"
	FileWireguardUpdate     = "/etc/wireguard/update"
	DefaultInfName          = "wg0"
)

type Config struct {
	HostInterface Host
	Peers         []Peer
}
