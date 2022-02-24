package wireguard

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

const WireguardConfigurationInterfaceTemplate = "[Interface]\nAddress = %s\nListenPort = %d\nPrivateKey = %s\n"
const WireguardConfigurationPeerTemplate = "[Peer]\nPublicKey = %s\nAllowedIPs = %s\nEndpoint = %s\n"

func TestParseConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Error("test harness expects linux")
	}
	configString := strings.Builder{}

	configString.WriteString(fmt.Sprintf(WireguardConfigurationInterfaceTemplate, "100.64.0.1", 51820, "base64_privatekey"))
	configString.WriteString(fmt.Sprintf(WireguardConfigurationPeerTemplate, "base64_publickey1", "100.64.0.2/32, 10.244.1.0/16", "10.240.0.4:51820"))
	configString.WriteString(fmt.Sprintf(WireguardConfigurationPeerTemplate, "base64_publickey2", "100.64.0.3/32, 10.244.2.0/16", "10.240.0.7:51820"))

	err := os.WriteFile("/tmp/wireguardtest.conf", []byte(configString.String()), 0644)
	if err != nil {
		t.Error(err)
	}
	cfg, err := ParseConfFile("/tmp/wireguardtest.conf")
	if err != nil {
		t.Error(err)
	}

	if cfg.HostInterface.Address != "100.64.0.1" {
		t.Errorf("host did not have right address. expected 100.64.0.1 got %s", cfg.HostInterface.Address)
	}

	if len(cfg.Peers) != 2 {
		t.Errorf("peers len does not equal expected value 2, got %d", len(cfg.Peers))
	}

	if cfg.Peers[0].AllowedIPs[0] != "100.64.0.2/32" || cfg.Peers[0].AllowedIPs[1] != "10.244.1.0/16" {
		t.Errorf("wrong allowed IPs")
	}

	os.Remove("/tmp/wireguardtest.conf")
}
