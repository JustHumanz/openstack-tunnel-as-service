package config

import "time"

var (
	KnowNumberPort = map[int]string{ // TODO
		22:   "ssh",
		80:   "http",
		3306: "mysql",
		8080: "tcp",
	}
	NgrokTunnelMetadata      = "ngrok_endpoint_%v"
	CloudflareTunnelMetadata = "cloudflare_endpoint_%v"
)

const (
	TunnelName     = "OpenStack_vm"
	TCPTimeout     = 1 * time.Minute
	ConnectionWait = 10 * time.Second
)

var (
	CFconfig = "config.yaml"
)

var (
	TunnelData = "TunnelsData.json"
)

func GetKnowPort(portNum int) string {
	if KnowNumberPort[portNum] == "" {
		return KnowNumberPort[8080]
	}
	return KnowNumberPort[portNum]
}
