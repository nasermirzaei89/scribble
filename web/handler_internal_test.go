package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeReturnToPath(t *testing.T) {
	t.Parallel()

	tt := []struct {
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
		{
			name:     "backslash-based absolute url is rejected",
			input:    "/\\evil.com",
			expected: "/",
		},
		{
			name:     "percent-encoded backslash-based absolute url is rejected",
			input:    "/%5C%5Cevil.com",
			expected: "/",
		},
		{
			name:     "CRLF in path is rejected",
			input:    "/foo\r\nLocation:https://evil.com",
			expected: "/",
		},
		{
			name:     "percent-encoded CRLF in path is rejected",
			input:    "/foo%0d%0aLocation:https://evil.com",
			expected: "/",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := sanitizeReturnToPath(tc.input)

			assert.Equal(t, tc.expected, result)
		})
	}
}
