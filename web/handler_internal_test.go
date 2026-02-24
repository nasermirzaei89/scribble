package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeReturnToPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty defaults to root",
			input:    "",
			expected: "/",
		},
		{
			name:     "relative path is allowed",
			input:    "/p/123",
			expected: "/p/123",
		},
		{
			name:     "relative path with query is allowed",
			input:    "/p/123?tab=comments",
			expected: "/p/123?tab=comments",
		},
		{
			name:     "relative path with fragment is allowed",
			input:    "/p/123#comments",
			expected: "/p/123#comments",
		},
		{
			name:     "missing leading slash is rejected",
			input:    "p/123",
			expected: "/",
		},
		{
			name:     "absolute url is rejected",
			input:    "https://evil.com",
			expected: "/",
		},
		{
			name:     "protocol relative url is rejected",
			input:    "//evil.com",
			expected: "/",
		},
		{
			name:     "triple slash is rejected",
			input:    "///evil.com",
			expected: "/",
		},
		{
			name:     "absolute url text as local path is allowed",
			input:    "/https://evil.com",
			expected: "/https://evil.com",
		},
		{
			name:     "double slash in local path is allowed",
			input:    "/foo//bar",
			expected: "/foo//bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizeReturnToPath(tt.input)

			assert.Equal(t, tt.expected, result)
		})
	}
}
