package comments

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortener_Process(t *testing.T) {
	cs := &Shortener{MaxLen: 100, TabLen: 4}

	src, err := os.ReadFile("testdata/comments.go")
	require.NoError(t, err)

	result := cs.Process(src)

	expectedContent, err := os.ReadFile("testdata/comments.go.golden")
	require.NoError(t, err)

	assert.Equal(t, string(expectedContent), string(result))
}
