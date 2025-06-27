package tunnel

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	tunnelConfig "github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

func (i *VmTunnel) SetCloudFlare(vmPort string, ips map[string]any) error {

	svcPort := tunnelConfig.ServiceID[vmPort]

	vmIPs := fmt.Sprintf("%v", ips)
	activeEndpoint, err := pkg.FindVMactiveIP(vmIPs, svcPort)
	if err != nil {
		return err
	}

	id := strings.Split(i.VMID, "-")[0]
	prefix := i.TunnelProvider.CF.SubDomainPrefix[vmPort]
	domain := i.TunnelProvider.CF.Domain
	sub := strings.Join([]string{id, prefix}, "-")
	vmDns := fmt.Sprintf("%v.%v", sub, domain)
	vmend := fmt.Sprintf("%v://%v", vmPort, activeEndpoint)

	log.Printf("Start vm tunneling with CloudFlare, name=%v id=%v svc=%v hostname=%v", i.VMname, i.VMID, activeEndpoint, vmDns)
	ctx, cancel := context.WithCancel(context.Background())
	err = i.TunnelProvider.CF.AddCFIngress(vmDns, vmend)
	if err != nil {
		return err
	}

	log.Printf("Create DNS Records, name=%v id=%v", i.VMname, i.VMID)
	err = i.TunnelProvider.CF.AddTunnelDNS(sub)
	if err != nil {
		return err
	}

	res := "https://" + vmDns

	VMEndpointMap := map[string]any{
		"address": strings.Split(activeEndpoint, ":")[0],
		"port": func() int {
			num, err := strconv.Atoi(strings.Split(activeEndpoint, ":")[1])
			if err != nil {
				log.Fatalln(err)
			}
			return num
		}(),
		"WellKnownPorts": vmPort,
	}

	TunnelEndpointMap := map[string]any{
		"address": vmDns,
		"port":    443,
	}

	i.VMSvc = append(i.VMSvc, VmSvc{
		CtxCancel:      cancel,
		Ctx:            ctx,
		ActiveIP:       vmIPs,
		VMEndpoint:     VMEndpointMap,
		TunnelEndpoint: TunnelEndpointMap,
	})

	endpoint := fmt.Sprintf("cf_endpoint_%v", vmPort)
	log.Printf("Update vm property, name=%v id=%v key=%v value=%v", i.VMname, i.VMID, endpoint, res)
	r := servers.UpdateMetadata(ctx, &i.OSCmpClient, i.VMID, servers.MetadataOpts{endpoint: res})
	if r.Err != nil {
		return err
	}

	return nil
}
