package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		limit    int
		array    []string
		expected [][]string
	}{
		{2, []string{"a", "b", "c", "d"}, [][]string{{"a", "b"}, {"c", "d"}}},
		{3, []string{"a", "b", "c", "d"}, [][]string{{"a", "b", "c"}, {"d"}}},
		{2, []string{"a", "b", "c"}, [][]string{{"a", "b"}, {"c"}}},
		{5, []string{"a", "b", "c"}, [][]string{{"a", "b", "c"}}},
		{-2, []string{"a", "b", "c"}, [][]string{{"a", "b"}, {"c"}}},
		{0, []string{"a", "b", "c"}, [][]string{{"a", "b", "c"}}},
	}

	for _, testCase := range testCases {
		assert.Equal(t, testCase.expected, split(testCase.array, testCase.limit))
	}
}
