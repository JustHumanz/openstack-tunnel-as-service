package tunnel

import (
	"fmt"
	"strings"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
)

func (sv *InstanceService) CreateCFSVC() string {
	prefix := sv.InstanceEndpoint.PortName
	return fmt.Sprintf("%s://%s", prefix, sv.InstanceEndpoint.Endpoint) //it's should become ssh://127.0.0.1:22
}

// Register new SVC into CloudFlare tunnel
func (InsTun *InstanceTunnel) AddCFTunnel(Prov provider.Provider) error {
	CFProvider := Prov.CF
	id := strings.Split(InsTun.InstanceID, "-")[0]
	domain := CFProvider.Domain
	for i := range InsTun.SVC {
		newSVC := InsTun.SVC[i]
		if newSVC.InstanceEndpoint.TunnelEndpoint != nil {
			continue
		}

		InstanceEP := newSVC.InstanceEndpoint
		prefix := InstanceEP.PortName
		sub := strings.Join([]string{id, prefix}, "-")
		vmDns := fmt.Sprintf("%v.%v", sub, domain)
		CFService := newSVC.CreateCFSVC()

		Log.Infof("Start vm tunneling with CloudFlare, name=%v id=%v svc=%v hostname=%v", InsTun.InstanceName, InsTun.InstanceID, InstanceEP.Endpoint, vmDns)
		err := CFProvider.AddCFIngress(vmDns, CFService)
		if err != nil {
			return err
		}
		Log.Infof("Create DNS Records, name=%v id=%v subdomain=%v", InsTun.InstanceName, InsTun.InstanceID, sub)
		err = CFProvider.AddTunnelDNS(sub)
		if err != nil {
			return err
		}

		InsTun.SVC[i].InstanceEndpoint.TunnelEndpoint = &Svc{
			Port:     443,
			PortName: prefix,
			Endpoint: vmDns,
		}
	}

	return nil
}

// Stop the cloudflare tunneling by Target vm endpoint or all tunneling if target targetSVC is nill
func (InsTun *InstanceTunnel) DeleteCFTunnel(targetSVC InstanceService, Prov provider.Provider) error {
	if (targetSVC == InstanceService{}) {
		for index, svc := range InsTun.SVC {
			InsTun.SVC = append(InsTun.SVC[:index], InsTun.SVC[index+1:]...)
			err := Prov.CF.StopCFIngress(svc.CreateCFSVC())
			if err != nil {
				return err
			}
			// TODO: Add func to delete the dns record
		}
	} else {
		for index, svc := range InsTun.SVC {
			if svc == targetSVC {
				InsTun.SVC = append(InsTun.SVC[:index], InsTun.SVC[index+1:]...)
				err := Prov.CF.StopCFIngress(svc.CreateCFSVC())
				if err != nil {
					return err
				}
				// TODO: Add func to delete the dns record

			}
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
