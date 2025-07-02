package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/db"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

var (
	tunnelVMs         = tunnel.TunnelData{}
	cloudflaredBin    = flag.String("cf", "/usr/bin/cloudflared", "The binary of cloudflared")
	cloudflaredDomain = flag.String("domain", "example.com", "The Domain of your cloudflare")
)

func init() {
	config.ServiceID = map[string]int{ // TODO
		"ssh":   22,
		"http":  80,
		"https": 443,
		"mysql": 3306,
	}

	log.SetPrefix("INFO: ")
	log.SetFlags(log.Ldate | log.Ltime)

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
		log.Println("Check CF tunnel")
		if !tunnelVMs.TunProvider.CF.CheckCFTunnel() {
			log.Println("OpenStack Tunnel not found, Create new CF tunnel")
			tunnelVMs.TunProvider.CF.CreateCFTunnel()

		}

		tunnelVMs.InitCFAPI()
		tunnelVMs.InitCFTunnel()
	} else {
		tunnelVMs.TunProvider = provider.Provider{
			NG: provider.Ngrok{
				Active:     true,
				StaticURLs: false, // https://dashboard.ngrok.com/tcp-addresses
			},
		}
		log.Println("Tunnel as service has ben started, init ngrok tunnel")
		tunnelVMs.InitNGCtx()
	}
}

func main() {
	log.Println("Starting tunnel as service")
	// create a scheduler
	s, err := gocron.NewScheduler()
	if err != nil {
		// handle error
		log.Fatalln(err)
	}

	// add a job to the scheduler
	_, err = s.NewJob(
		gocron.DurationJob(
			1*time.Minute,
		),
		gocron.NewTask(
			checkNewVMs,
		),
	)
	if err != nil {
		// handle error
		log.Fatalln(err)
	}

	// add a job to the scheduler
	_, err = s.NewJob(
		gocron.DurationJob(
			5*time.Minute,
		),
		gocron.NewTask(
			checkTunnelVMs,
		),
	)
	if err != nil {
		// handle error
		log.Fatalln(err)
	}

	// start the scheduler
	s.Start()

	//TODO: Create API
	select {}
}

func checkNewVMs() {
	log.Println("Start checking vms with tunnel property")
	ctx := context.Background()
	computeClient := pkg.InitComputeClient(ctx)
	allPages, err := servers.List(computeClient, servers.ListOpts{}).AllPages(ctx)
	if err != nil {
		log.Fatal(err)
	}

	vms, err := servers.ExtractServers(allPages)
	if err != nil {
		log.Fatal(err)
	}

	for _, vm := range vms {
		metaData := vm.Metadata["tunnel"]

		// If the vm already in list we should skip it
		if !tunnelVMs.GetVMTun(vm.ID) && metaData != "" {
			newTunnelVM := tunnel.VmTunnel{
				VMname: vm.Name,
				VMID:   vm.ID,
			}

			log.Printf("Found vm with tunnel property, name=%v id=%v", vm.Name, vm.ID)

			listSvc := strings.Split(metaData, ",")
			err := newTunnelVM.SetVMSvc(listSvc, vm.Addresses)
			if err != nil {
				log.Fatal(err)
			}

			if tunnelVMs.TunProvider.NG.Active {
				err := newTunnelVM.SetNgrok(tunnelVMs.TunProvider.NG)
				if err != nil {
					log.Fatal(err)
				}

				tunEndpoint := newTunnelVM.GetTunnelEndpoints()
				for i, val := range tunEndpoint {
					key := fmt.Sprintf(config.NgrokTunnelMetadata, listSvc[i])
					err := pkg.UpdateCmpProperty(computeClient, vm, key, val)
					if err != nil {
						log.Fatal(err)
					}
				}

			} else if tunnelVMs.TunProvider.CF.Active {
				err := newTunnelVM.SetCloudFlare(tunnelVMs.TunProvider.CF, true)
				if err != nil {
					log.Fatal(err)
				}

				tunEndpoint := newTunnelVM.GetTunnelEndpoints()
				for i, val := range tunEndpoint {
					key := fmt.Sprintf(config.CloudflareTunnelMetadata, listSvc[i])
					err := pkg.UpdateCmpProperty(computeClient, vm, key, val)
					if err != nil {
						log.Fatal(err)
					}
				}
			} else {
				log.Fatalln("tunnel provider not found")
			}

			tunnelVMs.AppendTunnels([]tunnel.VmTunnel{newTunnelVM})
		}
	}

	db.SaveTunnels(tunnelVMs.Tunnels)

}

func checkTunnelVMs() {
	log.Println("Check all vms with ngrok tunnel metadata")
	computeClient := pkg.InitComputeClient(context.Background())
	Prov := tunnelVMs.TunProvider
	NG := Prov.NG
	CF := Prov.CF

	for index, tunnelVM := range tunnelVMs.Tunnels {
		vm := servers.Get(context.Background(), computeClient, tunnelVM.VMID)
		if vm.Err != nil {
			if NG.Active {
				log.Printf("Server not found, delete all ngrok tunnel, name=%v id=%v", tunnelVM.VMname, tunnelVM.VMID)
				tunnelVM.StopNgrok(NG, "")
				tunnelVMs.RemoveTunnelsByIndex(index)
				continue
			} else if CF.Active {
				log.Printf("Server not found, delete all cloudflare tunnel, name=%v id=%v", tunnelVM.VMname, tunnelVM.VMID)
				err := tunnelVM.StopCloudFlare(CF, "")
				if err != nil {
					log.Fatal(err)
				}
				tunnelVMs.RemoveTunnelsByIndex(index)
				continue
			}
		}

		vmServer, err := vm.Extract()
		if err != nil {
			log.Fatal(err)
		}

		var tunnelSvc []string
		for key := range vmServer.Metadata {
			if strings.HasPrefix(key, "cloudflare") || strings.HasPrefix(key, "ngrok") {
				tunnelProperty := strings.Split(key, "_")
				tunnelSvc = append(tunnelSvc, tunnelProperty[len(tunnelProperty)-1])
			}
		}

		log.Printf("Check VM with tunnel property, name=%v id=%v", vmServer.Name, vmServer.ID)
		err = tunnelVM.CheckRemovedSvc(tunnelSvc, Prov, computeClient, vmServer)
		if err != nil {
			log.Fatal(err)
		}

		err = tunnelVM.CheckUpdatedSvc(tunnelSvc, Prov, computeClient, vmServer)
		if err != nil {
			log.Fatal(err)
		}
	}

	db.SaveTunnels(tunnelVMs.Tunnels)
}
