package graph

import (
	"bytes"
	"os"
	"testing"

	"github.com/dave/dst/decorator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDot(t *testing.T) {
	file, err := os.ReadFile("testdata/sample01.go")
	require.NoError(t, err)

	node, err := decorator.Parse(file)
	require.NoError(t, err)

	out := &bytes.Buffer{}

	err = CreateDot(node, out)
	require.NoError(t, err)

	expected, err := os.ReadFile("testdata/sample01.dot")
	require.NoError(t, err)

	assert.Equal(t, string(bytes.TrimSpace(expected)), out.String())
}
