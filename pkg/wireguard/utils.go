package wireguard

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	HostLabel       = "[Interface]"
	PeerLabel       = "[Peer]"
	KeyValuePattern = `^(\S+)[\s]+=[\s]+(.+)$`
)

func ParseConfigFile(fileName string) (WireguardConfiguration, error) {
	config := WireguardConfiguration{}
	fd, err := os.Open(fileName)
	if err != nil {
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
