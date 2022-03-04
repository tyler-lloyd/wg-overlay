package wireguard

import (
	"bufio"
	"errors"
	"io/fs"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.zx2c4.com/wireguard/wgctrl"
	"k8s.io/klog/v2"
)

const (
	HostLabel       = "[Interface]"
	PeerLabel       = "[Peer]"
	KeyValuePattern = `^(\S+)[\s]+=[\s]+(.+)$`
)

func GetConfig(deviceName string) (*Config, error) {
	ief, err := net.InterfaceByName(deviceName)
	if err != nil {
		return nil, err
	}

	addrs, err := ief.Addrs()
	if err != nil {
		return nil, err
	}

	var wireguardIP net.IP
	for _, addr := range addrs {
		if wireguardIP = addr.(*net.IPNet).IP.To4(); wireguardIP != nil {
			break
		}
	}

	c, err := wgctrl.New()
	if err != nil {
		return nil, err
	}

	dev, err := c.Device(deviceName)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		HostInterface: Host{
			PrivateKey: dev.PrivateKey.String(),
			PublicKey:  dev.PublicKey.String(),
			ListenPort: dev.ListenPort,
		},
	}
	if wireguardIP != nil {
		cfg.HostInterface.Address = wireguardIP.String()
	} else {
		klog.Warning("wireguard IP not set")
	}

	cfg.Peers = make([]Peer, 0)
	for _, peer := range dev.Peers {
		knownPeer := Peer{
			PublicKey: peer.PublicKey.String(),
			Endpoint:  peer.Endpoint.String(),
		}

		knownPeer.AllowedIPs = make([]string, 0)
		for _, ipNet := range peer.AllowedIPs {
			knownPeer.AllowedIPs = append(knownPeer.AllowedIPs, ipNet.String())
		}

		cfg.Peers = append(cfg.Peers, knownPeer)
	}

	return cfg, nil
}
func ParseConfFile(fileName string) (Config, error) {
	config := Config{}
	fd, err := os.Open(fileName)
	if errors.Is(err, fs.ErrNotExist) {
		return config, nil
	} else if err != nil {
		return config, err
	}
	defer fd.Close()
	scanner := bufio.NewScanner(fd)
	var curr string
	var prev string
	r, _ := regexp.Compile(KeyValuePattern)
	for scanner.Scan() {
		prev = curr
		curr = scanner.Text()
		if prev == HostLabel {
			hostInterface := Host{}
			loop := 0
			for curr != PeerLabel && curr != HostLabel && (loop < 1 || scanner.Scan()) {
				loop++
				curr = scanner.Text()
				groups := r.FindStringSubmatch(curr)
				if groups == nil {
					continue
				}
				k, v := groups[1], groups[2]
				switch k {
				case "PrivateKey":
					hostInterface.PrivateKey = v
				case "Address":
					hostInterface.Address = v
				case "ListenPort":
					i, _ := strconv.Atoi(v)
					hostInterface.ListenPort = i
				}
			}
			config.HostInterface = hostInterface
		}

		if prev == PeerLabel {
			peer := Peer{}
			loop := 0
			for curr != PeerLabel && curr != HostLabel && (loop < 1 || scanner.Scan()) {
				loop++
				groups := r.FindStringSubmatch(curr)
				curr = scanner.Text() // [Peer]
				if groups == nil {
					continue
				}

				k, v := groups[1], groups[2]
				switch k {
				case "PublicKey":
					peer.PublicKey = v
				case "AllowedIPs":
					v = strings.ReplaceAll(v, " ", "")
					peer.AllowedIPs = strings.Split(v, ",")
				case "Endpoint":
					peer.Endpoint = v
				}
			}
			config.Peers = append(config.Peers, peer)
		}
	}
	return config, nil
}
