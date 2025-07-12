package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/db"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
	Log "github.com/sirupsen/logrus"
)

var (
	tunnelVMs = tunnel.TunnelData{}
)

type SchedulerOps struct {
	ServerListOps       servers.ListOpts
	CheckVMSInterval    time.Duration
	CheckTunnelInterval time.Duration
}

func (i *SchedulerOps) StartScheduler() error {
	Log.Info("Starting scheduler")

	// create a scheduler
	s, err := gocron.NewScheduler()
	if err != nil {
		return err
	}

	// add a job to the scheduler
	_, err = s.NewJob(
		gocron.DurationJob(
			i.CheckVMSInterval,
		),
		gocron.NewTask(
			checkNewVMs,
			i.ServerListOps,
		),
	)
	if err != nil {
		// handle error
		return err
	}

	// add a job to the scheduler
	_, err = s.NewJob(
		gocron.DurationJob(
			i.CheckTunnelInterval,
		),
		gocron.NewTask(
			checkTunnelVMs,
		),
	)
	if err != nil {
		return err
	}

	// start the scheduler
	s.Start()

	return nil
}

func checkNewVMs(ops servers.ListOpts) {
	Log.Info("Start checking vms with tunnel property")
	ctx := context.Background()
	computeClient := pkg.InitComputeClient(ctx)
	allPages, err := servers.List(computeClient, ops).AllPages(ctx)
	if err != nil {
		Log.Error(err)
		return
	}

	vms, err := servers.ExtractServers(allPages)
	if err != nil {
		Log.Error(err)
		return
	}

	lenTunTmp := len(tunnelVMs.Tunnels)

	for _, vm := range vms {
		metaData := vm.Metadata["tunnel"]

		// If the vm already in list we should skip it
		if !tunnelVMs.GetVMTun(vm.ID) && metaData != "" {
			newTunnelVM := tunnel.VmTunnel{
				VMname: vm.Name,
				VMID:   vm.ID,
			}

			Log.Infof("Found vm with tunnel property, name=%v id=%v", vm.Name, vm.ID)

			listSvc := strings.Split(metaData, ",")
			err := newTunnelVM.SetVMSvc(listSvc, vm.Addresses)
			if err != nil {
				Log.Error(err)
				continue
			}

			if tunnelVMs.TunProvider.NG.Active {
				err := newTunnelVM.SetNgrok(tunnelVMs.TunProvider.NG)
				if err != nil {
					Log.Error(err)
					continue
				}

				tunEndpoint := newTunnelVM.GetTunnelEndpoints()
				for i, val := range tunEndpoint {
					key := fmt.Sprintf(config.NgrokTunnelMetadata, listSvc[i])
					err := pkg.UpdateCmpProperty(computeClient, vm, key, val)
					if err != nil {
						Log.Error(err)
						continue
					}
				}

			} else if tunnelVMs.TunProvider.CF.Active {
				err := newTunnelVM.SetCloudFlare(tunnelVMs.TunProvider.CF, true)
				if err != nil {
					Log.Error(err)
					continue
				}

				tunEndpoint := newTunnelVM.GetTunnelEndpoints()
				for i, val := range tunEndpoint {
					key := fmt.Sprintf(config.CloudflareTunnelMetadata, listSvc[i])
					err := pkg.UpdateCmpProperty(computeClient, vm, key, val)
					if err != nil {
						Log.Error(err)
						continue
					}
				}
			} else {
				Log.Fatal("tunnel provider not found")
			}

			tunnelVMs.AppendTunnels([]tunnel.VmTunnel{newTunnelVM})
		}
	}

	if lenTunTmp != len(tunnelVMs.Tunnels) {
		db.SaveTunnels(tunnelVMs.Tunnels)
	}

}

func checkTunnelVMs() {
	Log.Info("Check all vms with ngrok tunnel metadata")
	computeClient := pkg.InitComputeClient(context.Background())
	Prov := tunnelVMs.TunProvider
	NG := Prov.NG
	CF := Prov.CF

	updateDB := false
	for index, tunnelVM := range tunnelVMs.Tunnels {
		vm := servers.Get(context.Background(), computeClient, tunnelVM.VMID)
		if vm.Err != nil {
			if NG.Active {
				Log.Infof("Server not found, delete all ngrok tunnel, name=%v id=%v", tunnelVM.VMname, tunnelVM.VMID)
				tunnelVM.StopNgrok(NG, "")
				tunnelVMs.RemoveTunnelsByIndex(index)
				continue
			} else if CF.Active {
				Log.Infof("Server not found, delete all cloudflare tunnel, name=%v id=%v", tunnelVM.VMname, tunnelVM.VMID)
				err := tunnelVM.StopCloudFlare(CF, "")
				if err != nil {
					Log.Error(err)
					continue
				}
				tunnelVMs.RemoveTunnelsByIndex(index)
				continue
			}
		}

		vmServer, err := vm.Extract()
		if err != nil {
			Log.Error(err)
			continue
		}

		var tunnelSvc []string
		for key := range vmServer.Metadata {
			if strings.HasPrefix(key, "cloudflare") || strings.HasPrefix(key, "ngrok") {
				tunnelProperty := strings.Split(key, "_")
				tunnelSvc = append(tunnelSvc, tunnelProperty[len(tunnelProperty)-1])
			}
		}

		Log.Infof("Check VM with tunnel property, name=%v id=%v", vmServer.Name, vmServer.ID)
		removedSvc, err := tunnelVM.CheckRemovedSvc(tunnelSvc, Prov, computeClient, vmServer)
		if err != nil {
			Log.Error(err)
			continue
		}

		updatedSvc, err := tunnelVM.CheckUpdatedSvc(tunnelSvc, Prov, computeClient, vmServer)
		if err != nil {
			Log.Error(err)
			continue
		}

		if removedSvc != nil || updatedSvc != nil {
			updateDB = true
		}
	}

	if updateDB {
		db.SaveTunnels(tunnelVMs.Tunnels)
	}

}
