package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	tunnelConfig "github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

var (
	tunnelVMs         []tunnel.VmTunnel
	re                *regexp.Regexp
	provider          pkg.Provider
	cloudflaredBin    = flag.String("cf", "/usr/bin/cloudflared", "The binary of cloudflared")
	cloudflaredDomain = flag.String("domain", "example.com", "The Domain of your cloudflare")
)

func init() {
	tunnelConfig.ServiceID = map[string]int{ // TODO
		"ssh":   22,
		"http":  80,
		"https": 443,
		"mysql": 3306,
	}
	re = regexp.MustCompile(`(?m)(ngrok|cf)_endpoint_.+`)

	log.SetPrefix("INFO: ")
	log.SetFlags(log.Ldate | log.Ltime)

	if os.Getenv("CLOUDFLARE_API_KEY") != "" {
		flag.Parse()

		provider = pkg.Provider{
			CF: pkg.CloudFlare{
				CloudflaredPath: *cloudflaredBin,
				Domain:          *cloudflaredDomain,
				SubDomainPrefix: map[string]string{ //TODO
					"ssh": "ssh",
				},
				Active: true,
			},
		}
		log.Println("Check CF tunnel")
		if !provider.CF.CheckCFTunnel() {
			log.Println("OpenStack Tunnel not found, Create new CF tunnel")
			provider.CF.CreateCFTunnel()
		}

		provider.CF.InitAPI()
		provider.CF.InitTunnel()
	} else {
		provider.NG.Active = true
	}
}

func main() {
	ctx := context.Background()
	computeClient := tunnel.InitComputeClient(ctx)

	// Test struct
	/*
		ctxTest, cancelTest := context.WithCancel(context.Background())
		tunnelVMs = append(tunnelVMs, openStack.VmTunnel{
			VMname:      "ubuntu",
			VMID:        "eb7964e5-b601-413a-8e53-89951ed613fc",
			OSCmpClient: *computeClient,
			VMSvc: []openStack.VmSvc{
				openStack.VmSvc{
					CtxCancel: cancelTest,
					Ctx:       ctxTest,
					ActiveIP:  "172.16.18.245",
					VMEndpoint: map[string]any{
						"address":        "172.16.18.245",
						"port":           22,
						"WellKnownPorts": "ssh",
					},
				},
			},
		})
	*/

	// create a scheduler
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Add new job")
	// add a job to the scheduler
	_, err = s.NewJob(
		gocron.DurationJob(
			1*time.Minute,
		),
		gocron.NewTask(
			func() {
				log.Println("Searching new vms with tunnel metadata")
				allPages, err := servers.List(computeClient, servers.ListOpts{}).AllPages(ctx)
				if err != nil {
					log.Fatal(err)
				}

				vms, err := servers.ExtractServers(allPages)
				if err != nil {
					log.Fatal(err)
				}

				for _, vm := range vms {
					VMMetaData := vm.Metadata["tunnel"]
					tunnelVMSvc := strings.Split(VMMetaData, ",")

					//Check the vm with exsisting tunnel
					isNewVM := true
					for key := range vm.Metadata {
						if re.MatchString(key) {
							isNewVM = false
							break
						}
					}

					//New vm with tunnel property
					if VMMetaData != "" && isNewVM {
						newTunnelVM := tunnel.VmTunnel{
							VMname:         vm.Name,
							VMID:           vm.ID,
							OSCmpClient:    *computeClient,
							TunnelProvider: provider,
						}

						log.Printf("Found vm with tunnel property, name=%v id=%v", vm.Name, vm.ID)

						for _, vmPort := range tunnelVMSvc {
							if tunnelConfig.ServiceID[vmPort] != 0 {
								if provider.CF.Active {
									err := newTunnelVM.SetCloudFlare(vmPort, vm.Addresses)
									if err != nil {
										log.Fatalln(err)
									}
								} else {
									err := newTunnelVM.SetNgrok(vmPort, vm.Addresses)
									if err != nil {
										log.Fatalln(err)
									}
								}

							} else {
								log.Printf(fmt.Sprintf("Unsupported %v endpoint", vmPort))
							}
						}

						tunnelVMs = append(tunnelVMs, newTunnelVM)
					}
				}

			},
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(
			1*time.Minute,
		),
		gocron.NewTask(
			checkTunnelVMs,
		),
	)
	s.Start()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			// sig is a ^C, handle it
			log.Printf("captured %v, stopping profiler and exiting..", sig)
			if provider.NG.Active {
				for _, tunnelVM := range tunnelVMs {
					for _, VMSvc := range tunnelVM.VMSvc {
						v := "ngrok_endpoint_" + VMSvc.VMEndpoint["WellKnownPorts"].(string)
						log.Printf("Stop ngrok tunnel&Remove tunnel property from vm, name=%v id=%v svc=%v", tunnelVM.VMname, tunnelVM.VMID, v)
						tunnelVM.RemoveTunnelMetadata(v)
						VMSvc.CtxCancel()
						defer VMSvc.Ctx.Done()

					}
				}

			}

			pprof.StopCPUProfile()
			os.Exit(0)

		}
	}()

	select {}
}

func checkTunnelVMs() {
	log.Println("Check all vms with ngrok tunnel metadata")
	for i, tunnelVM := range tunnelVMs {
		vm := servers.Get(context.Background(), &tunnelVM.OSCmpClient, tunnelVM.VMID)
		if vm.Err != nil {
			log.Printf("Server not found, delete all ngrok tunnel, name=%v id=%v", tunnelVM.VMname, tunnelVM.VMID)
			tunnelVM.DeleteAllNgrokTunnel()
			tunnelVMs = tunnel.RemoveByIndex(tunnelVMs, i)
			continue
		}

		vmServer, err := vm.Extract()
		if err != nil {
			log.Fatal(err)
		}

		VMMetaData := vmServer.Metadata["tunnel"]
		tunnelVMSvc := strings.Split(VMMetaData, ",")

		var ngrokSvc []string
		for key := range vmServer.Metadata {
			if re.MatchString(key) && VMMetaData != "" {
				endpoint := strings.Split(key, "_")
				ngrokSvc = append(ngrokSvc, endpoint[len(endpoint)-1])
			}
		}

		if ngrokSvc == nil {
			continue
		}

		log.Printf("Check VM with ngrok tunnel, name=%v id=%v", vmServer.Name, vmServer.ID)
		diff := pkg.Difference(ngrokSvc, tunnelVMSvc)
		if diff != nil {
			log.Printf("Existing VM remove some endpoint, name=%v id=%v removed svc=%v", vmServer.Name, vmServer.ID, diff)
			for _, v := range diff {
				for svcIndex, vmSvc := range tunnelVMs[i].VMSvc {
					if vmSvc.VMEndpoint["port"] == tunnelConfig.ServiceID[v] {
						log.Printf("Stop ngrok tunnel, name=%v id=%v svc=%v", vmServer.Name, vmServer.ID, v)
						tunnelVMs[i].RemoveByIndex(svcIndex)
						log.Printf("Delete ngrok tunnel from vm property, name=%v id=%v svc=%v", vmServer.Name, vmServer.ID, v)
						tunnelVMs[i].RemoveTunnelMetadata("ngrok_endpoint_" + v)
					}
				}
			}
		}

		RevDiff := pkg.Difference(tunnelVMSvc, ngrokSvc)
		if RevDiff != nil {
			log.Printf("Existing VM update some endpoint, name=%v id=%v new svc=%v", vmServer.Name, vmServer.ID, RevDiff)
			for _, v := range RevDiff {
				//Starting new ngrok tunnel
				err := tunnelVMs[i].SetNgrok(v, vmServer.Addresses)
				if err != nil {
					log.Fatalln(err)
				}
			}
		}
	}
}
