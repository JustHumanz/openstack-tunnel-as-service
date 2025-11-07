package tunnel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceTunnel_GetInstanceMetadata(t *testing.T) {
	insTun := InstanceTunnel{
		SVC: []InstanceService{
			{
				InstanceEndpoint: Svc{
					Port: 22,
				},
			},
			{
				InstanceEndpoint: Svc{
					Port: 80,
				},
			},
		},
	}

	result := insTun.GetInstanceMetadata()
	expected := []string{"22", "80"}
	assert.Equal(t, expected, result)
}

func TestInstanceTunnel_GetInstanceMetadata_Empty(t *testing.T) {
	insTun := InstanceTunnel{
		SVC: []InstanceService{},
	}

	result := insTun.GetInstanceMetadata()
	assert.Equal(t, []string{}, result)
}

func TestTunnelData_UpdateTunnelData(t *testing.T) {
	tunData := TunnelData{
		Tunnels: []InstanceTunnel{
			{
				InstanceID: "id1",
				SVC: []InstanceService{
					{
						InstanceEndpoint: Svc{Port: 22},
					},
				},
			},
			{
				InstanceID: "id2",
			},
		},
	}

	new := InstanceTunnel{
		InstanceID: "id1",
		SVC: []InstanceService{
			{
				InstanceEndpoint: Svc{Port: 22},
			},
			{
				InstanceEndpoint: Svc{Port: 80},
			},
		},
	}

	tunData.UpdateTunnelData(new)

	assert.Equal(t, new, tunData.Tunnels[0])
	assert.Equal(t, "id2", tunData.Tunnels[1].InstanceID)
}

func TestTunnelData_GetVMTun(t *testing.T) {
	tunData := TunnelData{
		Tunnels: []InstanceTunnel{
			{InstanceID: "id1"},
			{InstanceID: "id2"},
		},
	}

	result := tunData.GetVMTun("id1")
	assert.NotNil(t, result)
	assert.Equal(t, "id1", result.InstanceID)

	result = tunData.GetVMTun("nonexistent")
	assert.Nil(t, result)
}

func TestTunnelData_RemoveTun(t *testing.T) {
	tunData := TunnelData{
		Tunnels: []InstanceTunnel{
			{InstanceID: "id1"},
			{InstanceID: "id2"},
			{InstanceID: "id3"},
		},
	}

	tunData.RemoveTun(&InstanceTunnel{InstanceID: "id2"})

	expected := []InstanceTunnel{
		{InstanceID: "id1"},
		{InstanceID: "id3"},
	}
	assert.Equal(t, expected, tunData.Tunnels)
}
