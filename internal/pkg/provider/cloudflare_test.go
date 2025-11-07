package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudFlare_CFTunnelCerd(t *testing.T) {
	cf := CloudFlare{
		TunnelID: "test-tunnel-id",
	}

	result, err := cf.CFTunnelCerd()
	assert.NoError(t, err)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".cloudflared", "test-tunnel-id.json")
	assert.Equal(t, expected, result)
}

func TestTunnelConfig_RemoveIngress(t *testing.T) {
	tc := TunnelConfig{
		Ingress: []Ingress{
			{Service: "service1"},
			{Service: "service2"},
			{Service: "service3"},
		},
	}

	tc.RemoveIngress(1)

	expected := []Ingress{
		{Service: "service1"},
		{Service: "service3"},
	}
	assert.Equal(t, expected, tc.Ingress)
}

func TestTunnelConfig_RemoveIngress_First(t *testing.T) {
	tc := TunnelConfig{
		Ingress: []Ingress{
			{Service: "service1"},
			{Service: "service2"},
		},
	}

	tc.RemoveIngress(0)

	expected := []Ingress{
		{Service: "service2"},
	}
	assert.Equal(t, expected, tc.Ingress)
}

func TestTunnelConfig_RemoveIngress_Last(t *testing.T) {
	tc := TunnelConfig{
		Ingress: []Ingress{
			{Service: "service1"},
			{Service: "service2"},
		},
	}

	tc.RemoveIngress(1)

	expected := []Ingress{
		{Service: "service1"},
	}
	assert.Equal(t, expected, tc.Ingress)
}
