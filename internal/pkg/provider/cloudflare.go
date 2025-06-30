package provider

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
)

// Check if the tunnel openstack already created
func (i *CloudFlare) CheckCFTunnel() bool {
	cmd := exec.Command(i.CloudflaredPath, "tunnel", "list", "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}

	tunnelList := []map[string]interface{}{}

	if err := json.Unmarshal(output, &tunnelList); err != nil {
		log.Fatalln(err)
	}

	for _, tunnel := range tunnelList {
		if tunnel["name"].(string) == config.TunnelName {
			i.TunnelID = tunnel["id"].(string)
			i.TunnelName = tunnel["name"].(string)
			CerdPath := i.CFTunnelCerd()
			if _, err := os.Stat(CerdPath); err != nil {
				log.Println("Tunnel credential not found, creating tunnel credential", CerdPath)
				cmd := exec.Command(i.CloudflaredPath, "tunnel", "token", "--cred-file", CerdPath, config.TunnelName)
				cmd.Stderr = os.Stderr
				_, err = cmd.Output()
				if err != nil {
					log.Fatalln(err)
				}
			}

			return true
		}
	}

	return false
}

// If the tunnel not yet created need to create it first
func (i *CloudFlare) CreateCFTunnel() {
	cmd := exec.Command(i.CloudflaredPath, "tunnel", "create", "--output", "json", config.TunnelName)
	output, err := cmd.Output()
	if err != nil {
		log.Fatalln(err)
	}

	var tunnel TunnelCreate
	if err := json.Unmarshal(output, &tunnel); err != nil {
		log.Fatalln(err)
	}

	i.TunnelID = tunnel.ID
	i.TunnelName = tunnel.Name
}

func (i *CloudFlare) CFTunnelCerd() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%v/.cloudflared/%v.json", home, i.TunnelID)
}

func (i *CloudFlare) InitTunnel() error {
	tunnelCfg, err := ReadCloudFlareConfig()
	if err != nil && tunnelCfg.Tunnel == "" {
		log.Println(err)

		tunnelCfg = TunnelConfig{
			Tunnel:          i.TunnelID,
			CredentialsFile: i.CFTunnelCerd(),
			OriginRequest: struct {
				ConnectTimeout string "yaml:\"connectTimeout\""
			}{
				ConnectTimeout: "30s",
			},
			Ingress: []Ingress{
				{
					Service: "http_status:404",
				},
			},
		}
	}

	WriteCloudFlareConfig(tunnelCfg)

	log.Println("Validate config")
	err = i.ValidateCFcfg()
	if err != nil {
		return err
	}

	log.Printf("Starting %v", i.CloudflaredPath)
	err = i.StartCF()
	if err != nil {
		return err
	}

	return nil
}

func (i *CloudFlare) ValidateCFcfg() error {
	cmd := exec.Command(i.CloudflaredPath, "tunnel", "--config", config.CFconfig, "ingress", "validate")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// Start new cloudflared
func (i *CloudFlare) StartCF() error {

	cmd := exec.Command(i.CloudflaredPath, "tunnel", "--config", config.CFconfig, "run", config.TunnelName)
	err := cmd.Start()
	if err != nil {
		return err
	}

	i.CloudFlareCmd = cmd

	return nil
}

// Reload the cloudflared
func (i *CloudFlare) ReloadCF() error {
	log.Printf("Reloading %v", i.CloudflaredPath)
	err := i.CloudFlareCmd.Process.Kill()
	if err != nil {
		return err
	}

	return i.StartCF()
}

// Add the new ingress
func (i *CloudFlare) AddCFIngress(VMHostname, VMService string) error {
	tunconf, err := ReadCloudFlareConfig()
	if err != nil {
		return err
	}
	newIngress := []Ingress{
		{
			Hostname: VMHostname,
			Service:  VMService,
		},
	}

	tunconf.Ingress = append(newIngress, tunconf.Ingress...)

	WriteCloudFlareConfig(tunconf)

	err = i.ValidateCFcfg()
	if err != nil {
		log.Fatal(err)
	}

	return i.ReloadCF()
}

// Stop/Delete cf ingress
func (i *CloudFlare) StopCFIngress(VMService string) error {
	tunconf, err := ReadCloudFlareConfig()
	if err != nil {
		return err
	}
	for index, ingress := range tunconf.Ingress {
		if ingress.Service == VMService {
			tunconf.RemoveIngress(index)
		}
	}

	WriteCloudFlareConfig(tunconf)

	err = i.ValidateCFcfg()
	if err != nil {
		log.Fatal(err)
	}

	return i.ReloadCF()
}

func (i *TunnelConfig) RemoveIngress(index int) {
	i.Ingress = append(i.Ingress[:index], i.Ingress[index+1:]...)
}

type TunnelCreate struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	CreatedAt   time.Time     `json:"created_at"`
	DeletedAt   time.Time     `json:"deleted_at"`
	Connections []interface{} `json:"connections"`
	Token       string        `json:"token"`
}

type TunnelConfig struct {
	Tunnel          string `yaml:"tunnel"`
	CredentialsFile string `yaml:"credentials-file"`
	OriginRequest   struct {
		ConnectTimeout string `yaml:"connectTimeout"`
	} `yaml:"originRequest"`
	Ingress []Ingress
}

type Ingress struct {
	Hostname string `yaml:"hostname,omitempty"`
	Service  string `yaml:"service"`
}
