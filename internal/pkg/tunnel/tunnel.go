package tunnel

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

type TunnelData struct {
	TunProvider provider.Provider
	Tunnels     []VmTunnel
}

type VmTunnel struct {
	VMname string  `json:"VMName"`
	VMID   string  `json:"VMID"`
	VMSvc  []VmSvc `json:"VMSvc"`
}

type VmSvc struct {
	TunnelEndpoint map[string]any `json:"TunnelEndpoint"`
	VMEndpoint     map[string]any `json:"VMEndpoint"`
}

func (i *VmTunnel) RemoveSvcByIndex(index int) {
	i.VMSvc = append(i.VMSvc[:index], i.VMSvc[index+1:]...)
}

func (i *TunnelData) RemoveTunnelsByIndex(index int) {
	i.Tunnels = append(i.Tunnels[:index], i.Tunnels[index+1:]...)
}

func (i *TunnelData) AppendTunnels(newTunnel []VmTunnel) {
	i.Tunnels = append(i.Tunnels, newTunnel...)
}

func (i *TunnelData) GetVMTun(vmID string) bool {
	for _, v := range i.Tunnels {
		if v.VMID == vmID {
			return true
		}
	}
	return false
}

func (i *VmTunnel) GetVMSvc() []string {
	var vmSvcList []string
	for _, v := range i.VMSvc {
		vmSvcList = append(vmSvcList, v.VMEndpoint["WellKnownPorts"].(string))
	}
	return vmSvcList
}

func (i *VmTunnel) CheckRemovedSvc(newVMSvc []string, v provider.Provider, computeClient *gophercloud.ServiceClient, vm *servers.Server) {
	currentVMSvc := i.GetVMSvc()
	diff := pkg.Difference(newVMSvc, currentVMSvc)
	if diff != nil {
		log.Printf("Existing VM removed some tunnel property, name=%v id=%v removed svc=%v", i.VMname, i.VMID, diff)
		for _, removedSvc := range diff {
			removedPort := config.ServiceID[removedSvc]
			for _, svc := range i.VMSvc {
				if svc.VMEndpoint["port"].(int) == removedPort {
					svcEndpoint := svc.GetVMEndpoint()
					if v.NG.Active {
						log.Printf("Stop ngrok tunnel, name=%v id=%v svc=%v", vm.Name, vm.ID, svcEndpoint)
						v.NG.NgrokStop(svcEndpoint)
						key := fmt.Sprintf(config.NgrokTunnelMetadata, removedSvc)
						log.Printf("Delete ngrok tunnel from vm property, name=%v id=%v svc=%v property=%v", vm.Name, vm.ID, svcEndpoint, key)
						pkg.RemoveCmpProperty(computeClient, i.VMID, key)

					} else if v.CF.Active {
						log.Printf("Stop cloduflare tunnel, name=%v id=%v svc=%v", vm.Name, vm.ID, svcEndpoint)
						err := v.CF.StopCFIngress(svcEndpoint)
						if err != nil {
							log.Fatal(err)
						}

						key := fmt.Sprintf(config.CloudflareTunnelMetadata, removedSvc)
						log.Printf("Delete cloduflare tunnel from vm property, name=%v id=%v svc=%v property=%v", vm.Name, vm.ID, svcEndpoint, key)
						pkg.RemoveCmpProperty(computeClient, i.VMID, key)
					}
				}
			}
		}
	}
}

func (i *VmTunnel) CheckUpdatedSvc(newVMSvc []string, v provider.Provider, computeClient *gophercloud.ServiceClient, vm *servers.Server) {
	currentVMSvc := i.GetVMSvc()
	diff := pkg.Difference(currentVMSvc, newVMSvc)
	if diff != nil {
		log.Printf("Existing VM update some tunnel property, name=%v id=%v updated svc=%v", i.VMname, i.VMID, diff)
		err := i.SetVMSvc(diff, vm.Addresses)
		if err != nil {
			log.Fatal(err)
		}

		if v.NG.Active {
			i.SetNgrok(v.NG)
		} else if v.CF.Active {
			i.SetCloudFlare(v.CF, true)
		}
	}
}

func (i *VmTunnel) SetVMSvc(listSvc []string, ips map[string]any) error {
	vmIPs := fmt.Sprintf("%v", ips)
	for _, v := range listSvc {
		svcPort := config.ServiceID[v]
		if svcPort != 0 {
			activeIPaddr, err := pkg.FindVMactiveIP(vmIPs, svcPort)
			if err != nil {
				return err
			}

			i.VMSvc = append(i.VMSvc, VmSvc{
				VMEndpoint: map[string]any{
					"WellKnownPorts": v,
					"address":        activeIPaddr,
					"port":           svcPort,
				},
			})
		} else {
			return fmt.Errorf("unsupported %v endpoint", v)
		}
	}

	return nil

}

func (i *VmTunnel) GetVMEndpoints() []string {
	var endPointList []string
	for _, v := range i.VMSvc {
		endPointList = append(endPointList, v.GetVMEndpoint())
	}
	return endPointList
}

func (i *VmTunnel) GetTunnelEndpoints() []string {
	var endPointList []string
	for _, v := range i.VMSvc {
		endPointList = append(endPointList, v.GetTunnelEndpoint())
	}
	return endPointList
}

func (i *VmSvc) GetTunnelEndpoint() string {
	vmPort := i.TunnelEndpoint["port"]
	vmActiveIP := i.TunnelEndpoint["address"]

	return fmt.Sprintf("%v:%v", vmActiveIP, vmPort)
}

func (i *VmSvc) GetVMEndpoint() string {
	vmPort := i.VMEndpoint["port"]
	vmActiveIP := i.VMEndpoint["address"]
	return fmt.Sprintf("%v:%v", vmActiveIP, vmPort)
}
