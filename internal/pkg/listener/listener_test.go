package listener

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_getTunnelMetaData(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]interface{}
		expected []string
	}{
		{
			name: "with tunnel metadata",
			payload: map[string]interface{}{
				"metadata": map[string]interface{}{
					"tunnel": "22,80",
				},
			},
			expected: []string{"22", "80"},
		},
		{
			name: "no tunnel metadata",
			payload: map[string]interface{}{
				"metadata": map[string]interface{}{},
			},
			expected: nil,
		},
		{
			name: "empty tunnel",
			payload: map[string]interface{}{
				"metadata": map[string]interface{}{
					"tunnel": "",
				},
			},
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message{Payload: tt.payload}
			result := m.getTunnelMetaData()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMessage_getInstanceName(t *testing.T) {
	m := Message{
		Payload: map[string]interface{}{
			"instance_id":  "test-id",
			"display_name": "test-name",
		},
	}

	id, name := m.getInstanceName()
	assert.Equal(t, "test-id", id)
	assert.Equal(t, "test-name", name)
}

func TestMessage_isActive(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		expected bool
	}{
		{
			name:     "active",
			state:    "active",
			expected: true,
		},
		{
			name:     "not active",
			state:    "stopped",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message{
				Payload: map[string]interface{}{
					"state": tt.state,
				},
			}
			result := m.isActive()
			assert.Equal(t, tt.expected, result)
		})
	}
}
