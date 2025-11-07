package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetKnowPort(t *testing.T) {
	tests := []struct {
		name     string
		portNum  int
		expected string
	}{
		{
			name:     "known port ssh",
			portNum:  22,
			expected: "ssh",
		},
		{
			name:     "known port http",
			portNum:  80,
			expected: "http",
		},
		{
			name:     "known port mysql",
			portNum:  3306,
			expected: "mysql",
		},
		{
			name:     "known port tcp",
			portNum:  8080,
			expected: "tcp",
		},
		{
			name:     "unknown port",
			portNum:  9999,
			expected: "tcp", // defaults to 8080
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetKnowPort(tt.portNum)
			assert.Equal(t, tt.expected, result)
		})
	}
}
