package config

var (
	ServiceID                map[string]int
	NgrokTunnelMetadata      = "ngrok_endpoint_%v"
	CloudflareTunnelMetadata = "cloudflare_endpoint_%v"
)

const (
	CFconfig   = "config.yaml"
	TunnelName = "OpenStack_vm"
	TunnelData = "TunnelsData.json"
)
