package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"gopkg.in/yaml.v2"
)

const config = "config.yaml"
const tunnelName = "openstack"

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
		if tunnel["name"].(string) == tunnelName {
			i.TunnelID = tunnel["id"].(string)
			i.TunnelName = tunnel["name"].(string)
			CerdPath := i.CFTunnelCerd()
			if _, err := os.Stat(CerdPath); err != nil {
				log.Println("Tunnel credential not found, creating tunnel credential", CerdPath)
				cmd := exec.Command(i.CloudflaredPath, "tunnel", "token", "--cred-file", CerdPath, tunnelName)
				cmd.Stderr = os.Stderr
				output, err = cmd.Output()
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
	cmd := exec.Command(i.CloudflaredPath, "tunnel", "create", "--output", "json", tunnelName)
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

func (i *CloudFlare) InitTunnel() {
	tunnelCfg := TunnelConfig{
		Tunnel:          i.TunnelID,
		CredentialsFile: i.CFTunnelCerd(),
		OriginRequest: struct {
			ConnectTimeout string "yaml:\"connectTimeout\""
		}{
			ConnectTimeout: "30s",
		},
		Ingress: []Ingress{
			Ingress{
				Service: "http_status:404",
			},
		},
	}
	data, err := yaml.Marshal(&tunnelCfg)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Write config to config.yaml")
	err = os.WriteFile(config, data, 0644)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Validate config")
	err = i.ValidateCFcfg()
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Starting %v", i.CloudflaredPath)
	err = i.StartCF()
	if err != nil {
		log.Fatalln(err)
	}
}

func (i *CloudFlare) ValidateCFcfg() error {
	cmd := exec.Command(i.CloudflaredPath, "tunnel", "--config", config, "ingress", "validate")
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

	cmd := exec.Command(i.CloudflaredPath, "tunnel", "--config", config, "run", tunnelName)
	err := cmd.Start()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
	log.Printf("Read %v file", config)
	data, err := os.ReadFile(config)
	if err != nil {
		return err
	}

	var tunconf TunnelConfig
	err = yaml.Unmarshal(data, &tunconf)
	if err != nil {
		return err
	}

	newIngress := []Ingress{
		Ingress{
			Hostname: VMHostname,
			Service:  VMService,
		},
	}

	tunconf.Ingress = append(newIngress, tunconf.Ingress...)

	newData, err := yaml.Marshal(&tunconf)
	if err != nil {
		return err
	}

	log.Printf("Update %v file", config)
	if err := os.WriteFile(config, newData, 0644); err != nil {
		return err
	}

	err = i.ReloadCF()
	if err != nil {
		log.Fatal(err)
	}

	return i.ValidateCFcfg()
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
