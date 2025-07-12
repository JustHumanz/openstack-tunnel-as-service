package main

import (
	"flag"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/db"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/scheduler"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/sirupsen/logrus"
)

var (
	tunnelVMs         = tunnel.TunnelData{}
	cloudflaredBin    = flag.String("cf", "/usr/bin/cloudflared", "The binary of cloudflared")
	cloudflaredDomain = flag.String("domain", "example.com", "The Domain of your cloudflare")
	Log               = logrus.New()
)

func init() {
	config.ServiceID = map[string]int{ // TODO
		"ssh":   22,
		"http":  80,
		"https": 443,
		"mysql": 3306,
	}

	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	Log.SetOutput(os.Stdout)
	Log.SetLevel(logrus.InfoLevel)

	tunnelVMs.Tunnels = db.LoadTunnels()
	if os.Getenv("CLOUDFLARE_API_KEY") != "" {
		flag.Parse()

		tunnelVMs.TunProvider = provider.Provider{
			CF: provider.CloudFlare{
				CloudflaredPath: *cloudflaredBin,
				Domain:          *cloudflaredDomain,
				SubDomainPrefix: map[string]string{ //TODO
					"ssh": "ssh",
				},
				Active: true,
			},
		}
		Log.Info("Check CF tunnel")
		if !tunnelVMs.TunProvider.CF.CheckCFTunnel() {
			Log.Info("OpenStack Tunnel not found, Create new CF tunnel")
			tunnelVMs.TunProvider.CF.CreateCFTunnel()

		}

		tunnelVMs.InitCFAPI()
		tunnelVMs.InitCFTunnel()
	} else if os.Getenv("NGROK_AUTHTOKEN") != "" {
		tunnelVMs.TunProvider = provider.Provider{
			NG: provider.Ngrok{
				Active:     true,
				StaticURLs: false, // https://dashboard.ngrok.com/tcp-addresses
			},
		}
		Log.Info("Tunnel as service has ben started, init ngrok tunnel")
		tunnelVMs.InitNGCtx()
	} else {
		Log.Fatal("Provider not found")
	}
}

func main() {
	Log.Info("Starting tunnel as service scheduler")
	schedulerOps := scheduler.SchedulerOps{
		CheckVMSInterval:    1 * time.Minute,
		CheckTunnelInterval: 5 * time.Minute,
		ServerListOps:       servers.ListOpts{},
	}

	err := schedulerOps.StartScheduler()
	if err != nil {
		Log.Fatal("Failed to start scheduler: ", err)
	}
	//TODO: Create API
	select {}
}
