package tunnel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceService_CreateCFSVC(t *testing.T) {
	tests := []struct {
		name     string
		svc      InstanceService
		expected string
	}{
		{
			name: "ssh service",
			svc: InstanceService{
				InstanceEndpoint: Svc{
					PortName: "ssh",
					Endpoint: "192.168.1.1:22",
				},
			},
			expected: "ssh://192.168.1.1:22",
		},
		{
			name: "http service",
			svc: InstanceService{
				InstanceEndpoint: Svc{
					PortName: "http",
					Endpoint: "10.0.0.1:80",
				},
			},
			expected: "http://10.0.0.1:80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.svc.CreateCFSVC()
			assert.Equal(t, tt.expected, result)
		})
	}
}
