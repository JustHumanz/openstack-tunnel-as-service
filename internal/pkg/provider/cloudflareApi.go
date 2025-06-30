package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudflare/cloudflare-go/v4"
	"github.com/cloudflare/cloudflare-go/v4/dns"
	"github.com/cloudflare/cloudflare-go/v4/option"
	"github.com/cloudflare/cloudflare-go/v4/zones"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"gopkg.in/yaml.v2"
)

const argoTunnel = "cfargotunnel.com"

// Init Cloudflare API
func (i *CloudFlare) InitAPI() error {
	client := cloudflare.NewClient(
		option.WithAPIToken(os.Getenv("CLOUDFLARE_API_KEY")), // defaults to os.LookupEnv("CLOUDFLARE_API_TOKEN")
	)

	i.CFapi.Client = client
	API := i.CFapi.Client

	page, err := API.Zones.List(context.TODO(), zones.ZoneListParams{})
	if err != nil {
		return err
	}

	for _, v := range page.Result {
		if strings.EqualFold(v.Name, i.Domain) {
			i.CFapi.ZoneID = v.ID
		}
	}
	return nil
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

// TODO: Add deleting dns trough CF API
/*
	func (i *CloudFlare) DeletingTunnelDNS(dnsRec string) error {
		client := i.CFapi.Client
		Content := fmt.Sprintf("%v.%v", i.TunnelID, argoTunnel)

		return err
	}
*/

// Read cf config file
func ReadCloudFlareConfig() (TunnelConfig, error) {
	log.Printf("Read %v file", config.CFconfig)
	data, err := os.ReadFile(config.CFconfig)
	if err != nil {
		return TunnelConfig{}, err
	}

	var tunconf TunnelConfig
	err = yaml.Unmarshal(data, &tunconf)
	if err != nil {
		return TunnelConfig{}, err
	}

	return tunconf, nil
}

func WriteCloudFlareConfig(tunconf TunnelConfig) {
	log.Printf("Write %v file", config.CFconfig)
	newData, err := yaml.Marshal(&tunconf)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Update %v file", config.CFconfig)
	if err := os.WriteFile(config.CFconfig, newData, 0644); err != nil {
		log.Fatal(err)
	}
}
