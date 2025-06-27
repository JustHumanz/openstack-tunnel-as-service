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

func (i *VmTunnel) SetNgrok(vmPort string, ips map[string]any) error {

	svcPort := tunnelConfig.ServiceID[vmPort]

	vmIPs := fmt.Sprintf("%v", ips)
	activeEndpoint, err := pkg.FindVMactiveIP(vmIPs, svcPort)
	if err != nil {
		return err
	}

	log.Printf("Start vm tunneling with Ngrok, name=%v id=%v svc=%v", i.VMname, i.VMID, activeEndpoint)
	ctx, cancel := context.WithCancel(context.Background())
	ngrokRes, err := pkg.NgrokForwarder(ctx, activeEndpoint)
	if err != nil {
		return err
	}

	res := ngrokRes.URL().Host

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
		"address": strings.Split(res, ":")[0],
		"port": func() int {
			num, err := strconv.Atoi(strings.Split(res, ":")[1])
			if err != nil {
				log.Fatalln(err)
			}
			return num
		}(),
	}

	i.VMSvc = append(i.VMSvc, VmSvc{
		CtxCancel:      cancel,
		Ctx:            ctx,
		ActiveIP:       vmIPs,
		VMEndpoint:     VMEndpointMap,
		TunnelEndpoint: TunnelEndpointMap,
	})

	endpoint := fmt.Sprintf("ngrok_endpoint_%v", vmPort)
	log.Printf("Update vm property, name=%v id=%v key=%v value=%v", i.VMname, i.VMID, endpoint, res)
	r := servers.UpdateMetadata(ctx, &i.OSCmpClient, i.VMID, servers.MetadataOpts{endpoint: res})
	if r.Err != nil {
		return err
	}

	return nil
}

func (i *VmTunnel) DeleteAllNgrokTunnel() {
	for _, tunnel := range i.VMSvc {
		tunnel.CtxCancel()
	}
}
