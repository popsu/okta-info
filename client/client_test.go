package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGroupRuleExpression(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "3 Groups",
			input: `"isMemberOfAnyGroup("00g1lghmvirItveA14x7", "00g360hu5bfvaBHP84x7", "00g1l7ll9aGlqnSg24x7")"`,
			expected: []string{
				"00g1lghmvirItveA14x7",
				"00g360hu5bfvaBHP84x7",
				"00g1l7ll9aGlqnSg24x7",
			},
		},
		{
			name:  "1 Group",
			input: `isMemberOfAnyGroup("00gar7xacmKf3wNAt4x7")`,
			expected: []string{
				"00gar7xacmKf3wNAt4x7",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, parseGroupRuleExpression(tc.input))
		})
	}
}
