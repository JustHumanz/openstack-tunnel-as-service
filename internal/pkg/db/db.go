package db

import (
	"encoding/json"
	"os"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/justhumanz/openstack-tunnel-as-service/pkg"
)

var (
	Log = pkg.Log // Use the log from pkg/log.go
)

// saveTunnels saves the current tunnelVMs to a JSON file.
// It creates the file if it does not exist and overwrites it if it does.
func SaveTunnels(Data []tunnel.InstanceTunnel) error {
	Log.Infof("Save tunnel data into %v", config.TunnelData)
	jsonData, err := json.MarshalIndent(Data, "", "  ") // pretty-print
	if err != nil {
		return err
	}

	err = os.WriteFile(config.TunnelData, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}

func LoadTunnels() ([]tunnel.InstanceTunnel, error) {
	Data := []tunnel.InstanceTunnel{}
	if _, err := os.Stat(config.TunnelData); err != nil {
		err = SaveTunnels(Data) // Create file if it does not exist
		if err != nil {
			return nil, err
		}
	}

	rawData, err := os.ReadFile(config.TunnelData)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(rawData, &Data)
	if err != nil {
		return []tunnel.InstanceTunnel{}, err
	}

	return Data, nil
}
