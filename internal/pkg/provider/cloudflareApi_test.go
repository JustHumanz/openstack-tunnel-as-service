package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justhumanz/openstack-tunnel-as-service/internal/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func setupTestCFConfig(t *testing.T) string {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_config.yaml")
	config.CFconfig = testFile
	return testFile
}

func TestReadCloudFlareConfig(t *testing.T) {
	testFile := setupTestCFConfig(t)

	expected := TunnelConfig{
		Tunnel: "test-tunnel",
		Ingress: []Ingress{
			{Service: "http_status:404"},
		},
	}

	data, err := yaml.Marshal(expected)
	assert.NoError(t, err)
	err = os.WriteFile(testFile, data, 0644)
	assert.NoError(t, err)

	result, err := ReadCloudFlareConfig()
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestReadCloudFlareConfig_FileNotExist(t *testing.T) {
	testFile := setupTestCFConfig(t)

	os.Remove(testFile)

	result, err := ReadCloudFlareConfig()
	assert.Error(t, err)
	assert.Equal(t, TunnelConfig{}, result)
}

func TestWriteCloudFlareConfig(t *testing.T) {
	testFile := setupTestCFConfig(t)

	tc := TunnelConfig{
		Tunnel: "test-tunnel",
		Ingress: []Ingress{
			{Service: "http_status:404"},
		},
	}

	WriteCloudFlareConfig(tc)

	assert.FileExists(t, testFile)

	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)

	var result TunnelConfig
	err = yaml.Unmarshal(data, &result)
	assert.NoError(t, err)
	assert.Equal(t, tc, result)
}
