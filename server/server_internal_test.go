package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDomainsToHTTPSAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		domains  []string
		expected string
	}{
		{
			name:     "single domain",
			domains:  []string{"example.com"},
			expected: "https://example.com",
		},
		{
			name:     "multiple domains",
			domains:  []string{"example.com", "www.example.com"},
			expected: "https://example.com, https://www.example.com",
		},
		{
			name:     "no domains",
			domains:  []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := domainsToHTTPSAddress(tt.domains)
			assert.Equal(t, tt.expected, result)
		})
	}
}
