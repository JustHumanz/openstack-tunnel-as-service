package openStack

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	tunnelConfig "github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

type VmTunnel struct {
	VMname      string
	VMID        string
	VMSvc       []VmSvc
	OSCmpClient gophercloud.ServiceClient
}

type VmSvc struct {
	CtxCancel      context.CancelFunc
	Ctx            context.Context
	ActiveIP       string
	TunnelEndpoint map[string]any
	VMEndpoint     map[string]any
}

func (v *VmTunnel) RemoveByIndex(index int) {
	defer v.VMSvc[index].Ctx.Done()
	v.VMSvc[index].CtxCancel()

	v.VMSvc = append(v.VMSvc[:index], v.VMSvc[index+1:]...)
}

func InitComputeClient(ctx context.Context) *gophercloud.ServiceClient {
	authOptions, endpointOptions, tlsConfig, err := clouds.Parse()
	if err != nil {
		panic(err)
	}

	providerClient, err := config.NewProviderClient(ctx, authOptions, config.WithTLSConfig(tlsConfig))
	if err != nil {
		panic(err)
	}

	computeClient, err := openstack.NewComputeV2(providerClient, endpointOptions)
	if err != nil {
		panic(err)
	}
	return computeClient
}

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

func (i *VmTunnel) DeleteAllTunnel() {
	for _, tunnel := range i.VMSvc {
		tunnel.CtxCancel()
	}
}

func RemoveByIndex(s []VmTunnel, index int) []VmTunnel {
	return append(s[:index], s[index+1:]...)
}

func (i *VmTunnel) RemoveNgrokMetadata(metadata string) {
	r := servers.DeleteMetadatum(context.Background(), &i.OSCmpClient, i.VMID, metadata)
	if r.Err != nil {
		log.Fatal(r.Err)
	}
}
