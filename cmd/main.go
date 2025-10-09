package main

import (
	"flag"
	"os"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/api"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/db"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/scheduler"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
	"github.com/sirupsen/logrus"
)

var (
	tunnelVMs         = tunnel.TunnelData{}
	cloudflaredBin    = flag.String("cf", "/usr/bin/cloudflared", "The binary of cloudflared")
	cloudflaredDomain = flag.String("domain", "example.com", "The Domain of your cloudflare")
	apiPort           = flag.Int("port", 8080, "The port for the API server")
	Log               = pkg.Log // Use the log from pkg/log.go
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

	// Load existing tunnels from the database
	err := error(nil)
	tunnelVMs.Tunnels, err = db.LoadTunnels()
	if err != nil {
		Log.Error(err)
	}

	switch {
	case os.Getenv("CLOUDFLARE_API_KEY") != "":
		flag.Parse()

		tunnelVMs.TunProvider = provider.Provider{
			CF: provider.CloudFlare{
				CloudflaredPath: *cloudflaredBin,
				Domain:          *cloudflaredDomain,
				SubDomainPrefix: map[string]string{ //TODO
					"ssh":  "ssh",
					"http": "http",
				},
				Active: true,
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
		tunnelVMs.InitNGCtx()
	default:
		Log.Fatal("Provider not found")
	}
}

func main() {
	schedulerOps := scheduler.SchedulerOps{
		CheckVMSInterval:    1 * time.Minute,
		CheckTunnelInterval: 5 * time.Minute,
		ServerListOps:       servers.ListOpts{},
		TunnelVMs:           &tunnelVMs,
	}

	Log.Info("Starting scheduler")
	err := schedulerOps.StartScheduler()
	if err != nil {
		Log.Fatal("Failed to start scheduler: ", err)
	}

	apiOps := api.APIops{
		TunnelVMs:  &tunnelVMs,
		ListenPort: *apiPort,
	}

	Log.Info("Starting API server")
	// Start the API server
	apiOps.StartAPI()

}
