package provider

import (
	"context"
	"os/exec"

	"github.com/cloudflare/cloudflare-go/v4"
)

type Provider struct {
	CF CloudFlare
	NG Ngrok
}

type Ngrok struct {
	Active     bool
	StaticURLs bool
	NgrokCtx   []NgCtx
}

type NgCtx struct {
	VMendpoint string
	CtxCancel  context.CancelFunc
	Ctx        context.Context
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
