package tunnel

import (
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

// Register new SVC into CloudFlare tunnel
func (tunData *TunnelData) AddNGTunnel(InsTun *InstanceTunnel) error {
	NGProvider := tunData.TunProvider.NG
	for i := range InsTun.SVC {
		newSVC := InsTun.SVC[i]
		InstanceEP := newSVC.InstanceEndpoint
		NGEndpointType := "tcp://"
		if InstanceEP.PortName == "http" {
			NGEndpointType = "https://"
			InstanceEP.Endpoint = fmt.Sprintf("http://%s", InstanceEP.Endpoint)
		} else {
			InstanceEP.Endpoint = fmt.Sprintf("tcp://%s", InstanceEP.Endpoint)
		}

		Log.Infof("Start vm tunneling with Ngrok, name=%v id=%v svc=%v", InsTun.InstanceName, InsTun.InstanceID, InstanceEP.Endpoint)
		ngrokRes, err := NGProvider.NgrokForwarder(InstanceEP.Endpoint, NGEndpointType)
		if err != nil {
			return err
		}

		InsTun.SVC[i].InstanceEndpoint.TunnelEndpoint = &Svc{
			Port:     443,
			PortName: InstanceEP.PortName,
			Endpoint: ngrokRes.URL().Host,
		}
	}

	return nil
}

func (InsTun *InstanceTunnel) PurgeNGEndpoint(cmp *gophercloud.ServiceClient) {
	for _, svc := range InsTun.SVC {
		key := fmt.Sprintf(config.NgrokTunnelMetadata, svc.InstanceEndpoint.TunnelEndpoint.PortName)
		Log.Infof("Delete ngrok tunnel from vm property, name=%v id=%v svc=%v", InsTun.InstanceName, InsTun.InstanceID, key)
		pkg.RemoveCmpProperty(cmp, InsTun.InstanceName, key)
	}
}

func (tunData *TunnelData) InitNGCtx(cmp *gophercloud.ServiceClient) error {
	if tunData.TunProvider.NG.Active {
		if !tunData.TunProvider.NG.StaticURLs {
			Log.Infof("Ngrok static url is %v deleting all ngrok tunnels", tunData.TunProvider.NG.StaticURLs)
			for i := range tunData.Tunnels {
				instanceTun := tunData.Tunnels[i]
				instanceTun.PurgeNGEndpoint(cmp)
				err := tunData.AddNGTunnel(&instanceTun)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	return nil
}

// Stop the ngrok tunneling by Target vm endpoint or all tunneling if target vm endpoint is empty
func (InsTun *InstanceTunnel) DeleteNGTunnel(Prov provider.Provider) {
	for _, svc := range InsTun.SVC {
		Prov.NG.NgrokStop(svc.InstanceEndpoint.TunnelEndpoint.Endpoint)
		// TODO: Add func to delete the dns record
	}
}
