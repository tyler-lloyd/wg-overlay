package wireguard

import "net"

type Host struct {
	Address    *net.IPNet
	PrivateKey string
	ListenPort int
	SaveConfig *bool
}
