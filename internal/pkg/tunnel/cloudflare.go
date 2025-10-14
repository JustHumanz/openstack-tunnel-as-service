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

func (sv *InstanceService) CreateCFSubDNS(instanceID string) string {
	id := strings.Split(instanceID, "-")[0]
	prefix := sv.InstanceEndpoint.PortName
	sub := strings.Join([]string{id, prefix}, "-")
	return sub
}

// Register new SVC into CloudFlare tunnel
func (InsTun *InstanceTunnel) AddCFTunnel(Prov provider.Provider) error {
	CFProvider := Prov.CF
	domain := CFProvider.Domain
	for i := range InsTun.SVC {
		newSVC := InsTun.SVC[i]
		if newSVC.InstanceEndpoint.TunnelEndpoint != nil {
			continue
		}

		sub := newSVC.CreateCFSubDNS(InsTun.InstanceID)
		vmDns := fmt.Sprintf("%v.%v", sub, domain)
		CFService := newSVC.CreateCFSVC()

		Log.Infof("Start vm tunneling with CloudFlare, name=%v id=%v svc=%v hostname=%v", InsTun.InstanceName, InsTun.InstanceID, newSVC.InstanceEndpoint.Endpoint, vmDns)
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
			PortName: newSVC.InstanceEndpoint.PortName,
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

			err = Prov.CF.DeleteTunnelDNS(svc.CreateCFSubDNS(InsTun.InstanceID))
			if err != nil {
				return err
			}
		}
	} else {
		for index, svc := range InsTun.SVC {
			if svc == targetSVC {
				InsTun.SVC = append(InsTun.SVC[:index], InsTun.SVC[index+1:]...)
				err := Prov.CF.StopCFIngress(svc.CreateCFSVC())
				if err != nil {
					return err
				}

				err = Prov.CF.DeleteTunnelDNS(svc.CreateCFSubDNS(InsTun.InstanceID))
				if err != nil {
					return err
				}

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
