package tunnel

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

// Starting tunnel as ngrok backend
func (i *VmTunnel) SetNgrok(v provider.Ngrok) error {

	for index, svc := range i.VMSvc {
		vmEndpoint := svc.GetVMEndpoint()

		log.Printf("Start vm tunneling with Ngrok, name=%v id=%v svc=%v", i.VMname, i.VMID, vmEndpoint)
		ngrokRes, err := v.NgrokForwarder(vmEndpoint, nil)
		if err != nil {
			return err
		}

		res := ngrokRes.URL().Host
		i.VMSvc[index].TunnelEndpoint = map[string]any{
			"address": strings.Split(res, ":")[0],
			"port": func() int {
				num, err := strconv.Atoi(strings.Split(res, ":")[1])
				if err != nil {
					log.Fatalln(err)
				}
				return num
			}(),
		}
	}

	return nil
}

// Stop the ngrok tunneling by Target vm endpoint or all tunneling if target vm endpoint is empty
func (i *VmTunnel) StopNgrok(v provider.Ngrok, TvmEndpoint string) {
	for _, svc := range i.VMSvc {
		vmEndpoint := svc.GetVMEndpoint()
		if vmEndpoint == TvmEndpoint || TvmEndpoint == "" {
			v.NgrokStop(vmEndpoint)
		}
	}
}

func (i *TunnelData) InitNGCtx() {
	if i.TunProvider.NG.Active {
		if !i.TunProvider.NG.StaticURLs {
			log.Printf("Ngrok static url is %v deleting all ngrok tunnels", i.TunProvider.NG.StaticURLs)

			for index, tun := range i.Tunnels {
				for _, svc := range tun.VMSvc {
					ep := svc.GetTunnelEndpoint()
					key := fmt.Sprintf(config.NgrokTunnelMetadata, svc.VMEndpoint["WellKnownPorts"].(string))
					computeClient := pkg.InitComputeClient(context.Background())
					log.Printf("Delete ngrok tunnel from vm property, name=%v id=%v svc=%v property=%v", tun.VMname, tun.VMID, ep, key)
					pkg.RemoveCmpProperty(computeClient, tun.VMID, key)
				}

				tun.StopNgrok(i.TunProvider.NG, "")
				i.RemoveTunnelsByIndex(index)
			}
		} else {
			log.Printf("Ngrok static url is %v starting all ngrok tunnels", i.TunProvider.NG.StaticURLs)
			for _, tun := range i.Tunnels {
				for _, svc := range tun.VMSvc {
					vmEndpoint := svc.GetVMEndpoint()
					tunEndpoint := svc.GetTunnelEndpoint()
					log.Printf("Starting %v", tunEndpoint)

					_, err := i.TunProvider.NG.NgrokForwarder(vmEndpoint, &tunEndpoint)
					if err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
}
