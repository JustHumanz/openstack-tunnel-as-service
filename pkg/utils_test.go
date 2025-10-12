package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOpenStackIPs(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "single IP",
			input:    "192.168.1.1",
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "multiple IPs",
			input:    "192.168.1.1 10.0.0.1",
			expected: []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:     "with brackets",
			input:    "[192.168.1.1]",
			expected: []string{"192.168.1.1"},
		},
		{
			name:     "empty",
			input:    "",
			expected: []string{},
		},
		{
			name:     "no IP",
			input:    "no ip here",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseOpenStackIPs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDifference(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "no difference",
			a:        []string{"a", "b"},
			b:        []string{"a", "b"},
			expected: []string{},
		},
		{
			name:     "some difference",
			a:        []string{"a", "b", "c"},
			b:        []string{"b"},
			expected: []string{"a", "c"},
		},
		{
			name:     "empty a",
			a:        []string{},
			b:        []string{"a"},
			expected: []string{},
		},
		{
			name:     "empty b",
			a:        []string{"a"},
			b:        []string{},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Difference(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTestInstanceEP_InvalidAddress(t *testing.T) {
	result := TestInstanceEP("invalid:address")
	assert.False(t, result)
}

func TestFindVMactiveIP_NoActive(t *testing.T) {
	result, err := FindVMactiveIP("invalid", 9999)
	assert.Error(t, err)
	assert.Equal(t, "", result)
}
