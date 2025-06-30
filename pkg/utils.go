package pkg

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"regexp"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
)

const ipv4Regex = `\b(?:\d{1,3}\.){3}\d{1,3}\b`
const tcpTimeout = 3 * time.Second

func FindVMactiveIP(vmIps string, vmSvc int) (string, error) {
	ips := regexp.MustCompile(ipv4Regex).FindAllString(vmIps, -1)
	for _, ip := range ips {
		vmIp := fmt.Sprintf("%v:%v", ip, vmSvc)
		//fmt.Println("Try connection into:", vmIp)

		conn, err := net.DialTimeout("tcp", vmIp, tcpTimeout)
		if err != nil {
			//fmt.Println("TCP connection failed:", err)
			continue
		}

		defer conn.Close()
		return ip, nil
	}

	return "", errors.New("VM service unreachable")
}

func Difference(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range b {
		m[item] = true
	}

	var diff []string
	for _, item := range a {
		if !m[item] {
			diff = append(diff, item)
		}
	}
	return diff
}

func InitComputeClient(ctx context.Context) *gophercloud.ServiceClient {
	authOptions, endpointOptions, tlsConfig, err := clouds.Parse()
	if err != nil {
		panic(err)
	}

	providerClient, err := config.NewProviderClient(ctx, authOptions, config.WithTLSConfig(tlsConfig))
	if err != nil {
		panic(err)
	}

	computeClient, err := openstack.NewComputeV2(providerClient, endpointOptions)
	if err != nil {
		panic(err)
	}
	return computeClient
}

func UpdateCmpProperty(cmp *gophercloud.ServiceClient, vm servers.Server, key, value string) error {
	log.Printf("Update vm property, name=%v id=%v key=%v value=%v", vm.Name, vm.ID, key, value)
	r := servers.UpdateMetadata(context.Background(), cmp, vm.ID, servers.MetadataOpts{key: value})
	if r.Err != nil {
		return r.Err
	}

	return nil
}

func RemoveCmpProperty(cmp *gophercloud.ServiceClient, vmid, metadata string) {
	r := servers.DeleteMetadatum(context.Background(), cmp, vmid, metadata)
	if r.Err != nil {
		log.Fatal(r.Err)
	}
}
