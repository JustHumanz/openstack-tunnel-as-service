package pkg

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	OSConf "github.com/gophercloud/gophercloud/v2/openstack/config"
	"github.com/gophercloud/gophercloud/v2/openstack/config/clouds"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
)

const ipv4Regex = `\b(?:\d{1,3}\.){3}\d{1,3}\b`
const tcpTimeout = 1 * time.Minute

func ParseOpenStackIPs(fixed_ips any) []string {
	result := regexp.MustCompile(ipv4Regex).FindAllString(fmt.Sprintf("%v", fixed_ips), -1)
	if result == nil {
		return []string{}
	}
	return result
}

func TestInstanceEP(ep string) bool {
	for i := 0; i <= 5; i++ {
		Log.Infof("Connection checking. %d Attempting.", i)

		conn, err := net.DialTimeout("tcp", ep, config.TCPTimeout)
		if err != nil {
			Log.Error(err)
			time.Sleep(config.ConnectionWait)
			continue
		}

		defer conn.Close()
		return true
	}

	return false
}

func FindVMactiveIP(vmIps string, vmSvc int) (string, error) {
	ips := regexp.MustCompile(ipv4Regex).FindAllString(vmIps, -1)

	for i := 0; i <= 5; i++ {
		for _, ip := range ips {
			Log.Infof("Connection checking. %d Attempting.", i)
			vmIp := fmt.Sprintf("%v:%v", ip, vmSvc)

			conn, err := net.DialTimeout("tcp", vmIp, tcpTimeout)
			if err != nil {
				Log.Error(err)
				continue
			}

			defer conn.Close()
			return ip, nil
		}

		time.Sleep(1 * time.Second)
	}

	return "", errors.New("VM service unreachable")
}

func Difference(a, b []string) []string {
	m := make(map[string]bool)
	for _, item := range b {
		m[item] = true
	}

	diff := []string{}
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

	providerClient, err := OSConf.NewProviderClient(ctx, authOptions, OSConf.WithTLSConfig(tlsConfig))
	if err != nil {
		panic(err)
	}

	computeClient, err := openstack.NewComputeV2(providerClient, endpointOptions)
	if err != nil {
		panic(err)
	}
	return computeClient
}

func UpdateCmpProperty(cmp *gophercloud.ServiceClient, vmID, key, value string) error {
	Log.Infof("Update vm property, id=%v key=%v value=%v", vmID, key, value)
	r := servers.UpdateMetadata(context.Background(), cmp, vmID, servers.MetadataOpts{key: value})
	if r.Err != nil {
		return r.Err
	}

	return nil
}

func RemoveCmpProperty(cmp *gophercloud.ServiceClient, vmID, key string) error {
	Log.Infof("Remove vm property, id=%v key=%v", vmID, key)
	r := servers.DeleteMetadatum(context.Background(), cmp, vmID, key)
	if r.Err != nil {
		return r.Err
	}

	return nil
}

func GetInstanceDetails(computeClient *gophercloud.ServiceClient, InstanceID string) (*servers.Server, error) {
	vm := servers.Get(context.Background(), computeClient, InstanceID)
	if vm.Err != nil {
		return nil, vm.Err
	}

	vmServer, err := vm.Extract()
	if err != nil {
		return nil, err
	}

	return vmServer, nil
}

func StripScheme(s string) string {
	if i := strings.Index(s, "://"); i != -1 {
		return s[i+3:]
	}
	return s
}
