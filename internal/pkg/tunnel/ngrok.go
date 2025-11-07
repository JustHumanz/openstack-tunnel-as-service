package tunnel

import (
	"strconv"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
)

// Purge ngrok endpoint tunnel
func (InsTun *InstanceTunnel) PurgeNGEndpoint() error {
	for i := range InsTun.SVC {
		InsTun.SVC[i].InstanceEndpoint.TunnelEndpoint = nil
	}
	return nil
}

// Register new SVC into CloudFlare tunnel
func (InsTun *InstanceTunnel) AddNGTunnel(Prov *provider.Provider) error {
	for i := range InsTun.SVC {
		svc := &InsTun.SVC[i]
		ep := &svc.InstanceEndpoint

		if ep.TunnelEndpoint != nil {
			continue
		}

		var ngEndpointType, fullEndpoint string
		switch ep.PortName {
		case "http":
			ngEndpointType = "https://"
			fullEndpoint = "http://" + ep.Endpoint
		default:
			ngEndpointType = "tcp://"
			fullEndpoint = "tcp://" + ep.Endpoint
		}

		Log.Infof("Start vm tunneling with Ngrok, name=%v id=%v svc=%v", InsTun.InstanceName, InsTun.InstanceID, fullEndpoint)
		ngrokRes, err := Prov.NG.NgrokForwarder(fullEndpoint, ngEndpointType)
		if err != nil {
			return err
		}

		url := ngrokRes.URL()
		var portNum int
		if url.Scheme == "https" {
			portNum = 443
		} else {
			if p := url.Port(); p != "" {
				portNum, err = strconv.Atoi(p)
				if err != nil {
					return err
				}
			} else {
				portNum = 80 // fallback or default, adjust as needed
			}
		}

		ep.TunnelEndpoint = &Svc{
			Port:     portNum,
			PortName: ep.PortName,
			Endpoint: url.Host,
		}
	}
	return nil
}

func (tunData *TunnelData) InitNGCtx(cmp *gophercloud.ServiceClient) error {
	if tunData.TunProvider.NG.Active {
		if !tunData.TunProvider.NG.StaticURLs {
			Log.Infof("Ngrok static url is %v Reload all ngrok tunnels", tunData.TunProvider.NG.StaticURLs)
			for i := range tunData.Tunnels {
				instanceTun := tunData.Tunnels[i]
				instanceTun.PurgeNGEndpoint()

				err := instanceTun.AddNGTunnel(&tunData.TunProvider)
				if err != nil {
					return err
				}

				//Update the results of Tunnel Backend endpoint
				err = instanceTun.UpdateAllInstanceMetadata(cmp, tunData.TunProvider)
				if err != nil {
					return err
				}
			}
		} else {
			Log.Infof("Ngrok static url is %v starting all ngrok tunnels", tunData.TunProvider.NG.StaticURLs)
			for i := range tunData.Tunnels {
				instanceTun := tunData.Tunnels[i]
				err := instanceTun.AddNGTunnel(&tunData.TunProvider)
				if err != nil {
					return err
				}

				//Update the results of Tunnel Backend endpoint
				err = instanceTun.UpdateAllInstanceMetadata(cmp, tunData.TunProvider)
				if err != nil {
					return err
				}

			}
		}
		return nil
	}

	return nil
}

// Stop the ngrok tunneling by Target vm endpoint or all tunneling if targetSVC is nil
func (InsTun *InstanceTunnel) DeleteNGTunnel(targetSVC *InstanceService, Prov *provider.Provider) error {
	newSVC := InsTun.SVC[:0] // Reuse underlying array
	for _, svc := range InsTun.SVC {
		// If targetSVC is zero, remove all; else, remove only matching
		toDelete := (targetSVC == nil) || (svc == *targetSVC)
		if toDelete {
			Prov.NG.NgrokStop(svc.InstanceEndpoint.Endpoint)
		} else {
			newSVC = append(newSVC, svc)
		}
	}
	InsTun.SVC = newSVC
	return nil
}
