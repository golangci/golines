package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_isGenerated(t *testing.T) {
	testCases := []struct {
		desc   string
		file   string
		assert assert.BoolAssertionFunc
	}{
		{
			desc:   "license before comment about generated code",
			file:   "testdata/generated_license.go",
			assert: assert.True,
		},
		{
			desc:   "license before comment about generated code (star)",
			file:   "testdata/generated_license_star.go",
			assert: assert.True,
		},
		{
			desc:   "no generated code comment",
			file:   "testdata/not_generated.go",
			assert: assert.False,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(filepath.FromSlash(test.file))
			require.NoError(t, err)

			test.assert(t, isGenerated(content))
		})
	}
}
