package tunnel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/provider"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

var (
	Log = pkg.Log
)

type TunnelData struct {
	TunProvider provider.Provider
	Tunnels     []InstanceTunnel
}

type InstanceTunnel struct {
	InstanceName string            `json:"InstanceName"`
	InstanceID   string            `json:"InstanceID"`
	ActiveIP     string            `json:"ActiveIP"`
	SVC          []InstanceService `json:"Svc"`
}

type InstanceService struct {
	InstanceEndpoint Svc `json:"InstanceEndpoint"`
}

type Svc struct {
	Port           int
	PortName       string
	Endpoint       string
	TunnelEndpoint *Svc
}

// Update the tunnels data
func (tunData *TunnelData) UpdateTunnelData(new InstanceTunnel) {
	for i, v := range tunData.Tunnels {
		if v.InstanceID == new.InstanceID {
			tunData.Tunnels[i] = new
		}
	}
}

// Get the current metadata on of Instance
func (insTun *InstanceTunnel) GetInstanceMetadata() []string {
	metadata := []string{}
	for _, v := range insTun.SVC {
		metadata = append(metadata, strconv.Itoa(v.InstanceEndpoint.Port))
	}
	return metadata
}

func (insTun *InstanceTunnel) UpdateInstace(newMetadata []string, Prov *provider.Provider) ([]string, []string, error) {
	var removedEP, newEP []string
	oldMetadata := insTun.GetInstanceMetadata()
	removedTmp := pkg.Difference(oldMetadata, newMetadata)
	newTmp := pkg.Difference(newMetadata, oldMetadata)

	Log.Infof("Instance update existing tunnel, Removed svc=%v New Svc=%v", removedTmp, newTmp)
	if removedTmp != nil {
		for _, v := range insTun.SVC {
			for _, k := range removedTmp {
				if strconv.Itoa(v.InstanceEndpoint.Port) == k {
					removedEP = append(removedEP, v.InstanceEndpoint.PortName)

					switch {
					case Prov.CF.Active:
						err := insTun.DeleteCFTunnel(&v, Prov)
						if err != nil {
							return nil, nil, err
						}
					case Prov.NG.Active:
						err := insTun.DeleteNGTunnel(&v, Prov)
						if err != nil {
							return nil, nil, err
						}
					default:
						Log.Error("Invalid Provider")
					}
				}
			}
		}
	}

	if newTmp != nil {
		eps, err := insTun.ParseNewSVC(newTmp)
		if err != nil {
			return nil, nil, err
		}

		err = insTun.AddSVC(eps)
		if err != nil {
			return nil, nil, err
		}
		newEP = eps

		switch {
		case Prov.CF.Active:
			err = insTun.AddCFTunnel(Prov)
			if err != nil {
				return nil, nil, err
			}
		case Prov.NG.Active:
			err = insTun.AddNGTunnel(Prov)
			if err != nil {
				return nil, nil, err
			}
		default:
			Log.Error("Invalid Provider")
		}

	}

	return removedEP, newEP, nil
}

// Parse the New Tunnel to 127.0.0.1:22 and testing the connection
func (insTun *InstanceTunnel) ParseNewSVC(tunMetadata []string) ([]string, error) {
	var newSVCs []string
	for _, v := range tunMetadata {

		i, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}

		ep := fmt.Sprintf("%s:%d", insTun.ActiveIP, i)
		Log.Infof("Testing TCP Connection to %s", ep)
		if !pkg.TestInstanceEP(ep) {
			continue
		}

		newSVCs = append(newSVCs, ep)
	}

	return newSVCs, nil
}

// Add the new service from metadata into SVC struct
func (insTun *InstanceTunnel) AddSVC(eps []string) error {
	for _, ep := range eps {
		tmpep := strings.Split(ep, ":")
		PortNum, err := strconv.Atoi(tmpep[1])
		if err != nil {
			return err
		}
		PortName := config.GetKnowPort(PortNum)
		insTun.SVC = append(insTun.SVC, InstanceService{
			InstanceEndpoint: Svc{
				Port:     PortNum,
				PortName: PortName,
				Endpoint: ep,
			},
		})
	}

	return nil
}

func (tun *TunnelData) AddNewTun(newTun *InstanceTunnel, newTunMetaData []string) error {
	eps, err := newTun.ParseNewSVC(newTunMetaData)
	if err != nil {
		return err
	}

	err = newTun.AddSVC(eps)
	if err != nil {
		return err
	}

	switch {
	case tun.TunProvider.CF.Active:
		err = newTun.AddCFTunnel(&tun.TunProvider)
		if err != nil {
			return err
		}
	case tun.TunProvider.NG.Active:
		err = newTun.AddNGTunnel(&tun.TunProvider)
		if err != nil {
			return err
		}
	default:
		Log.Error("Invalid Provider")
	}

	return nil
}

func (tun *TunnelData) DeleteTun(deleteTun *InstanceTunnel) error {
	if tun.TunProvider.CF.Active {
		Log.Infof("Server not found, delete all cloudflare tunnel, name=%v id=%v", deleteTun.InstanceName, deleteTun.InstanceID)
		return deleteTun.DeleteCFTunnel(nil, &tun.TunProvider)

	} else if tun.TunProvider.NG.Active {
		Log.Infof("Server not found, delete all ngrok tunnel, name=%v id=%v", deleteTun.InstanceName, deleteTun.InstanceID)
		deleteTun.DeleteNGTunnel(nil, &tun.TunProvider)
	}

	return nil
}

func (instance *InstanceTunnel) UpdateAllInstanceMetadata(computeClient *gophercloud.ServiceClient, Prov provider.Provider) error {
	for _, svc := range instance.SVC {
		key := ""
		switch {
		case Prov.CF.Active:
			key = fmt.Sprintf(config.CloudflareTunnelMetadata, svc.InstanceEndpoint.PortName)

		case Prov.NG.Active:
			key = fmt.Sprintf(config.NgrokTunnelMetadata, svc.InstanceEndpoint.PortName)
		}
		err := pkg.UpdateCmpProperty(computeClient, instance.InstanceID, key, svc.InstanceEndpoint.TunnelEndpoint.Endpoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *InstanceTunnel) UpdateOneInstanceMetadata(computeClient *gophercloud.ServiceClient, Prov provider.Provider, targetEP string) error {
	for _, svc := range instance.SVC {
		if svc.InstanceEndpoint.Endpoint != targetEP {
			continue
		}

		key := ""
		switch {
		case Prov.CF.Active:
			key = fmt.Sprintf(config.CloudflareTunnelMetadata, svc.InstanceEndpoint.PortName)

		case Prov.NG.Active:
			key = fmt.Sprintf(config.NgrokTunnelMetadata, svc.InstanceEndpoint.PortName)
		}
		err := pkg.UpdateCmpProperty(computeClient, instance.InstanceID, key, svc.InstanceEndpoint.TunnelEndpoint.Endpoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *InstanceTunnel) DelAllInstanceMetadata(computeClient *gophercloud.ServiceClient, Prov provider.Provider) error {
	for _, svc := range instance.SVC {
		err := instance.DelOneInstanceMetadata(computeClient, Prov, svc.InstanceEndpoint.PortName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (instance *InstanceTunnel) DelOneInstanceMetadata(computeClient *gophercloud.ServiceClient, Prov provider.Provider, targetEP string) error {
	key := ""
	switch {
	case Prov.CF.Active:
		key = fmt.Sprintf(config.CloudflareTunnelMetadata, targetEP)

	case Prov.NG.Active:
		key = fmt.Sprintf(config.NgrokTunnelMetadata, targetEP)
	}
	err := pkg.RemoveCmpProperty(computeClient, instance.InstanceID, key)
	if err != nil {
		return err
	}
	return nil
}
func (tun *TunnelData) GetVMTun(instanceID string) *InstanceTunnel {
	for _, v := range tun.Tunnels {
		if v.InstanceID == instanceID {
			return &v
		}
	}
	return nil
}

// Remove the tunnel from the list
func (tun *TunnelData) RemoveTun(removedTun *InstanceTunnel) {
	for i := len(tun.Tunnels) - 1; i >= 0; i-- {
		if tun.Tunnels[i].InstanceID == removedTun.InstanceID {
			tun.Tunnels = append(tun.Tunnels[:i], tun.Tunnels[i+1:]...)
		}
	}
}
