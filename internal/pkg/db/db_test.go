package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/justhumanz/openstack-tunnel-as-service/internal/pkg/tunnel"
	"github.com/stretchr/testify/assert"
)

func setupTestFile(t *testing.T) string {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_tunnels.json")
	config.TunnelData = testFile
	return testFile
}

func TestSaveTunnels(t *testing.T) {
	testFile := setupTestFile(t)

	data := []tunnel.InstanceTunnel{
		{
			InstanceName: "test-vm-1",
			InstanceID:   "test-id-1",
			ActiveIP:     "192.168.1.1",
			SVC: []tunnel.InstanceService{
				{
					InstanceEndpoint: tunnel.Svc{
						Port:     22,
						PortName: "ssh",
						Endpoint: "192.168.1.1:22",
					},
				},
			},
		},
		{
			InstanceName: "test-vm-2",
			InstanceID:   "test-id-2",
			ActiveIP:     "10.0.0.1",
			SVC:          []tunnel.InstanceService{},
		},
	}

	err := SaveTunnels(data)
	assert.NoError(t, err)

	// Verify file exists and content
	assert.FileExists(t, testFile)

	loaded, err := LoadTunnels()
	assert.NoError(t, err)
	assert.Equal(t, data, loaded)
}

func TestLoadTunnels_FileDoesNotExist(t *testing.T) {
	testFile := setupTestFile(t)

	// Ensure file doesn't exist
	os.Remove(testFile)

	loaded, err := LoadTunnels()
	assert.NoError(t, err)
	assert.Equal(t, []tunnel.InstanceTunnel{}, loaded)

	// Verify empty file was created
	assert.FileExists(t, testFile)
}

func TestLoadTunnels_InvalidJSON(t *testing.T) {
	testFile := setupTestFile(t)

	// Write invalid JSON
	err := os.WriteFile(testFile, []byte("invalid json"), 0644)
	assert.NoError(t, err)

	loaded, err := LoadTunnels()
	assert.Error(t, err)
	assert.Equal(t, []tunnel.InstanceTunnel{}, loaded)
}
