package db

import (
	"encoding/json"
	"log"
	"os"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
)

type VmTunnelJson struct {
	VMname string         `json:"VMName"`
	VMID   string         `json:"VMID"`
	VMSvc  []tunnel.VmSvc `json:"VMSvc"`
}

// saveTunnels saves the current tunnelVMs to a JSON file.
// It creates the file if it does not exist and overwrites it if it does.
func SaveTunnels(Data []tunnel.VmTunnel) {
	log.Printf("Save tunnel data into %v", config.TunnelData)
	tunnelVMsJson := []VmTunnelJson{}
	for _, v := range Data {
		tunnelVMsJson = append(tunnelVMsJson, VmTunnelJson{
			VMname: v.VMname,
			VMID:   v.VMID,
			VMSvc:  v.VMSvc,
		})
	}

	jsonData, err := json.MarshalIndent(tunnelVMsJson, "", "  ") // pretty-print
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(config.TunnelData, jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func LoadTunnels() []tunnel.VmTunnel {
	var Data []tunnel.VmTunnel
	if _, err := os.Stat(config.TunnelData); err != nil {
		SaveTunnels(Data) // Create file if it does not exist
	}

	tunnelVMsJson := []VmTunnelJson{}
	rawData, err := os.ReadFile(config.TunnelData)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(rawData, &tunnelVMsJson)
	if err != nil {
		log.Fatal(err)
	}

	for index := range tunnelVMsJson {
		Data = append(Data, tunnel.VmTunnel{
			VMname: tunnelVMsJson[index].VMname,
			VMID:   tunnelVMsJson[index].VMID,
			VMSvc:  tunnelVMsJson[index].VMSvc,
		})

	}
	return Data
}
