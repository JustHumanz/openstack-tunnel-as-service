package pkg

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
)

const argoTunnel = "cfargotunnel.com"

// Init Cloudflare API
func (i *CloudFlare) InitAPI() {
	client := cloudflare.NewClient(
		option.WithAPIToken(os.Getenv("CLOUDFLARE_API_KEY")), // defaults to os.LookupEnv("CLOUDFLARE_API_TOKEN")
	)

	i.CFapi.Client = client
	API := i.CFapi.Client

	page, err := API.Zones.List(context.TODO(), zones.ZoneListParams{})
	if err != nil {
		panic(err.Error())
	}

	for _, v := range page.Result {
		if strings.EqualFold(v.Name, i.Domain) {
			i.CFapi.ZoneID = v.ID
		}
	}
}

func (i *CloudFlare) AddTunnelDNS(dnsRec string) error {
	client := i.CFapi.Client
	Content := fmt.Sprintf("%v.%v", i.TunnelID, argoTunnel)
	_, err := client.DNS.Records.New(context.Background(), dns.RecordNewParams{
		ZoneID: cloudflare.String(i.CFapi.ZoneID),
		Body: dns.CNAMERecordParam{
			Name:    cloudflare.String(dnsRec),
			Content: cloudflare.String(Content),
			Type:    cloudflare.Raw[dns.CNAMERecordType](dns.CNAMERecordTypeCNAME),
			Proxied: cloudflare.Bool(true),
			Comment: cloudflare.String("Created by openstack tunnel"),
		},
	})

	return err
}
