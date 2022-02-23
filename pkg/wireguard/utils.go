package wireguard

import (
	"os/exec"

	"k8s.io/klog/v2"
)

func ParseConfigFile(b []byte) WireguardConfiguration {
	s := string(b)

	klog.Info(s)

	return WireguardConfiguration{}
}

func Genkey() (string, error) {
	key, err := exec.Command("wg", "genkey").Output()
	if err != nil {
		return "", err
	}
	return string(key), nil
}

func Pubkey(privateKey string) (string, error) {
	cmd := "cat /etc/wireguard/privatekey | wg pubkey"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	klog.Info(string(out))
	return string(out), nil
}
