package pkg

import (
	"os/exec"

	"github.com/cloudflare/cloudflare-go/v4"
)

type Provider struct {
	CF CloudFlare
	NG Ngrok
}

type Ngrok struct {
	Active bool
}

type CloudFlare struct {
	Active          bool
	CloudflaredPath string
	Domain          string
	SubDomainPrefix map[string]string
	TunnelID        string
	TunnelName      string
	CloudFlareCmd   *exec.Cmd
	CFapi           API
}

type API struct {
	Client *cloudflare.Client
	ZoneID string
}
