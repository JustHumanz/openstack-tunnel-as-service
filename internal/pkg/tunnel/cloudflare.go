package tunnel

import (
	"fmt"
	"log"
	"strings"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
)

func (i *VmTunnel) SetCloudFlare(v provider.CloudFlare, dns bool) error {

	for index, svc := range i.VMSvc {
		knowPort := svc.VMEndpoint["WellKnownPorts"].(string)
		vmEndpointRaw := svc.GetVMEndpoint()
		vmEndpoint := fmt.Sprintf("%v://%v", knowPort, vmEndpointRaw)

		id := strings.Split(i.VMID, "-")[0]
		prefix := v.SubDomainPrefix[knowPort]
		domain := v.Domain
		sub := strings.Join([]string{id, prefix}, "-")
		vmDns := fmt.Sprintf("%v.%v", sub, domain)

		log.Printf("Start vm tunneling with CloudFlare, name=%v id=%v svc=%v hostname=%v", i.VMname, i.VMID, vmEndpoint, vmDns)
		err := v.AddCFIngress(vmDns, vmEndpoint)
		if err != nil {
			return err
		}

		if dns {
			log.Printf("Create DNS Records, name=%v id=%v subdomain=%v", i.VMname, i.VMID, sub)
			err = v.AddTunnelDNS(sub)
			if err != nil {
				return err
			}
		}

		i.VMSvc[index].TunnelEndpoint = map[string]any{
			"address": vmDns,
			"port":    443,
		}

	}
	return nil
}

// Stop the ngrok tunneling by Target vm endpoint or all tunneling if target vm endpoint is empty
func (i *VmTunnel) StopCloudFlare(v provider.CloudFlare, TvmEndpoint string) error {
	for _, svc := range i.VMSvc {
		vmEndpoint := svc.GetVMEndpoint()
		if vmEndpoint == TvmEndpoint || TvmEndpoint == "" {
			return v.StopCFIngress(vmEndpoint)
			// TODO: Add func to delete the dns record
		}
	}

	return nil
}

func (i *TunnelData) InitCFAPI() error {
	return i.TunProvider.CF.InitAPI()
}

func (i *TunnelData) InitCFTunnel() error {
	return i.TunProvider.CF.InitTunnel()
}
