package tunnel

import (
	"context"
	"log"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

type VmTunnel struct {
	VMname         string
	VMID           string
	VMSvc          []VmSvc
	OSCmpClient    gophercloud.ServiceClient
	TunnelProvider pkg.Provider
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

func RemoveByIndex(s []VmTunnel, index int) []VmTunnel {
	return append(s[:index], s[index+1:]...)
}

func (i *VmTunnel) RemoveTunnelMetadata(metadata string) {
	r := servers.DeleteMetadatum(context.Background(), &i.OSCmpClient, i.VMID, metadata)
	if r.Err != nil {
		log.Fatal(r.Err)
	}
}
