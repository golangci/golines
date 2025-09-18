package diff

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPretty(t *testing.T) {
	// For now, this just tests that the script runs without error
	_, err := Pretty(
		"test_path.txt",
		[]byte("line 1\nline 2"),
		[]byte("line 1\nline 2 modified"),
	)
	require.NoError(t, err)
}
