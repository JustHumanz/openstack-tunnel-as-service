package listener

import (
	"encoding/json"
	"errors"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/db"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	queueName      = "notifications.info"
	instanceCreate = "compute.instance.create.end"
	instanceDelete = "compute.instance.delete.end"
	instanceUpdate = "compute.instance.update"
	ActiveStatus   = "active"
)

var (
	Log = pkg.Log // Use the log from pkg/log.go
)

type ListenerOps struct {
	InstancesTun *tunnel.TunnelData
	Cmp          *gophercloud.ServiceClient
	AmqpURL      string
}

func (i *ListenerOps) StartListener() error {
	Log.Info("Starting OpenStack message queue listener")
	conn, err := amqp.Dial(i.AmqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	// Subscribing to QueueService1 for getting messages.
	messages, err := ch.Consume(
		queueName, // queue name
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no local
		false,     // no wait
		nil,       // arguments
	)
	if err != nil {
		return err
	}

	outputTmp := Oslo{}

	for message := range messages {
		json.Unmarshal(message.Body, &outputTmp)

		var m Message
		json.Unmarshal([]byte(outputTmp.Message), &m)
		switch m.EventType {

		//New VM created with tunnel metadata
		case instanceCreate:
			if m.isActive() {
				err := CreateTun(m, i.InstancesTun, i.Cmp)
				if err != nil {
					Log.Error(err)
				}
			}
		case instanceDelete:
			if !m.isActive() {
				err := DestroyTunnel(m, i.InstancesTun)
				if err != nil {
					Log.Error(err)
				}
			}
		case instanceUpdate:
			if m.isActive() && m.Payload["building"] == nil && m.Payload["old_task_state"] == nil {
				err := UpdateTunnel(m, i.InstancesTun, i.Cmp)
				if err != nil {
					Log.Error(err)
				}
			}
		default:
			continue
		}
	}
	return nil
}

func CreateTun(m Message, TunData *tunnel.TunnelData, computeClient *gophercloud.ServiceClient) error {
	instanceID, instanceName := m.getInstanceName()
	metadata := m.getTunnelMetaData()

	if metadata != nil {
		if TunData.GetVMTun(instanceID) == nil {
			Log.Infof("New Instance created with tunnel metadata, checking for tunnels, name=%v id=%v", instanceName, instanceID)
			vmIps := m.Payload["fixed_ips"]
			NewTunIP := pkg.ParseOpenStackIPs(vmIps)
			if NewTunIP == nil {
				return errors.New("invalid IPaddr")
			}

			NewTun := tunnel.InstanceTunnel{
				InstanceName: instanceName,
				InstanceID:   instanceID,
				ActiveIP:     NewTunIP[0],
			}

			//Add new tun into Tunnel Backend
			err := TunData.AddNewTun(&NewTun, metadata)
			if err != nil {
				return err
			}

			//Update the results of Tunnel Backend endpoint
			err = NewTun.UpdateAllInstanceMetadata(computeClient, TunData.TunProvider)
			if err != nil {
				return err
			}

			//Append the new tuns into db and save it
			TunData.Tunnels, err = db.LoadTunnels()
			if err != nil {
				return err
			}

			TunData.Tunnels = append(TunData.Tunnels, NewTun)
			err = db.SaveTunnels(TunData.Tunnels)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func DestroyTunnel(m Message, TunData *tunnel.TunnelData) error {
	instanceID, instanceName := m.getInstanceName()
	Log.Infof("Instance deleted, removing checking for tunnels, name=%v id=%v", instanceName, instanceID)

	DeletedVM := TunData.GetVMTun(instanceID)
	if DeletedVM != nil {
		err := TunData.DeleteTun(DeletedVM)
		if err != nil {
			return err
		}

		//Reload the Tunnel list from json file
		TunData.Tunnels, err = db.LoadTunnels()
		if err != nil {
			return err
		}

		//Remove the Index file
		TunData.RemoveTun(DeletedVM)
		err = db.SaveTunnels(TunData.Tunnels)
		if err != nil {
			return err
		}
	}

	return nil
}

func UpdateTunnel(m Message, TunData *tunnel.TunnelData, computeClient *gophercloud.ServiceClient) error {
	instanceID, instanceName := m.getInstanceName()
	metadata := m.getTunnelMetaData()
	ExistingVM := TunData.GetVMTun(instanceID)

	//Updating the existing service
	if metadata != nil && ExistingVM != nil {
		Log.Infof("Instance update existing tunnel, checking for tunnels, name=%v id=%v", instanceName, instanceID)
		removedEP, newEP, err := ExistingVM.UpdateInstace(metadata, &TunData.TunProvider)
		if err != nil {
			return err
		}

		for _, v := range removedEP {
			Log.Infof("Remove tunnel endpoint metadata, name=%v id=%v key=%v", instanceName, instanceID, v)
			err := ExistingVM.DelOneInstanceMetadata(computeClient, TunData.TunProvider, v)
			if err != nil {
				return err
			}
		}

		for _, v := range newEP {
			Log.Infof("Add new tunnel endpoint metadata, name=%v id=%v value=%v", instanceName, instanceID, v)
			err := ExistingVM.UpdateOneInstanceMetadata(computeClient, TunData.TunProvider, v)
			if err != nil {
				return err
			}
		}

		TunData.UpdateTunnelData(*ExistingVM)

		err = db.SaveTunnels(TunData.Tunnels)
		if err != nil {
			return err
		}

		//New Tunnel VM
	} else if metadata != nil && ExistingVM == nil {
		Log.Infof("Instance create new tunnel, checking for tunnels, name=%v id=%v", instanceName, instanceID)
		instanceDetails, err := pkg.GetInstanceDetails(computeClient, instanceID)
		if err != nil {
			return err
		}
		m.Payload["fixed_ips"] = instanceDetails.Addresses
		return CreateTun(m, TunData, computeClient)

		//Deleting all the Tunnel
	} else if metadata == nil && ExistingVM != nil {
		Log.Infof("Instance deleting all existing tunnel, checking for tunnels, name=%v id=%v", instanceName, instanceID)
		err := DestroyTunnel(m, TunData)
		if err != nil {
			return err
		}

		//Delete tunnel metadata on instances
		err = ExistingVM.DelAllInstanceMetadata(computeClient, TunData.TunProvider)
		if err != nil {
			return err
		}
	}

	return nil
}
