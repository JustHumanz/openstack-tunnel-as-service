package main

import (
	"context"
	"flag"
	"os"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/db"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/listener"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
	"github.com/sirupsen/logrus"
)

var (
	tunnelVMs         = tunnel.TunnelData{}
	cloudflaredBin    = flag.String("cf", "/usr/bin/cloudflared", "The binary of cloudflared")
	cloudflaredDomain = flag.String("domain", "example.com", "The Domain of your cloudflare")
	AmqpURL           = flag.String("amqpURL", "amqp://nova:rabbitmq@127.0.0.1:5672/nova", "nova amqp url transporter")
	Log               = pkg.Log // Use the log from pkg/log.go
	cmp               *gophercloud.ServiceClient
)

func init() {
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)
	flag.Parse()

	cmp = pkg.InitComputeClient(context.Background())
	// Load existing tunnels from the database
	err := error(nil)
	tunnelVMs.Tunnels, err = db.LoadTunnels()
	if err != nil {
		Log.Error(err)
	}

	switch {
	case os.Getenv("CLOUDFLARE_API_KEY") != "":

		tunnelVMs.TunProvider = provider.Provider{
			CF: provider.CloudFlare{
				CloudflaredPath: *cloudflaredBin,
				Domain:          *cloudflaredDomain,
				Active:          true,
			},
		}
		Log.Info("Check CF tunnel")

		tun, err := tunnelVMs.TunProvider.CF.CheckCFTunnel()
		if err != nil {
			Log.Fatal("Failed to check CF tunnel: ", err)
		}
		if !tun {
			Log.Info("OpenStack Tunnel not found, Create new CF tunnel")
			tunnelVMs.TunProvider.CF.CreateCFTunnel()
		}

		tunnelVMs.InitCFAPI()
		tunnelVMs.InitCFTunnel()
	case os.Getenv("NGROK_AUTHTOKEN") != "":
		tunnelVMs.TunProvider = provider.Provider{
			NG: provider.Ngrok{
				Active:     true,
				StaticURLs: false, // https://dashboard.ngrok.com/tcp-addresses
			},
		}
		Log.Info("Tunnel as service has ben started, init ngrok tunnel")
		tunnelVMs.InitNGCtx(cmp)
	default:
		Log.Fatal("Provider not found")
	}
}

func main() {
	listenerOps := listener.ListenerOps{
		InstancesTun: &tunnelVMs,
		Cmp:          cmp,
		AmqpURL:      *AmqpURL,
	}

	Log.Info("Starting amqp listener")
	err := listenerOps.StartListener()
	if err != nil {
		Log.Error(err)
	}

}
