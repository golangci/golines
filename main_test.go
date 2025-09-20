package main

import (
	"bytes"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testFiles = map[string]string{
	"test1.go": `package main

import "fmt"

func main() {
	fmt.Printf("%s %s %s %s %s %s", "argument1", "argument2", "argument3", "argument4", "argument5", "argument6")
}`,
	"test2.go": `package main

func main() {
	myMap := map[string]string{"key1": "value1", "key2": "value2", "key3": "value3", "key4": "value4", "key5", "value5"}
}`,
}

func Test_runner_run_dir(t *testing.T) {
	tmpDir := t.TempDir()

	writeTestFiles(t, testFiles, tmpDir)

	runner := NewRunner()
	runner.args = []string{tmpDir}

	s := newSequencer(1, os.Stdout, os.Stderr)

	runner.run(s)
	require.Equal(t, 0, s.GetExitCode())

	// Without writeOutput set to true, inputs should be unchanged
	for name, expected := range testFiles {
		path := filepath.Join(tmpDir, name)

		content, err := os.ReadFile(path)
		require.NoError(t, err, "Unexpected error reading test file")

		assert.Equal(
			t,
			strings.TrimSpace(expected),
			strings.TrimSpace(string(content)),
		)
	}

	runner.writeOutput = true

	runner.run(s)
	require.Equal(t, 0, s.GetExitCode())

	// Now, files should be modified in place
	for name, expected := range testFiles {
		path := filepath.Join(tmpDir, name)

		content, err := os.ReadFile(path)
		require.NoError(t, err, "Unexpected error reading test file")

		assert.NotEqual(
			t,
			strings.TrimSpace(expected),
			strings.TrimSpace(string(content)),
		)
	}
}

func Test_runner_run_filePaths(t *testing.T) {
	tmpDir := t.TempDir()

	runner := NewRunner()
	runner.writeOutput = true
	runner.args = append(runner.args, writeTestFiles(t, testFiles, tmpDir)...)

	s := newSequencer(1, os.Stdout, os.Stderr)

	runner.run(s)
	require.Equal(t, 0, s.GetExitCode())

	// Now, files should be modified in place
	for name, expected := range testFiles {
		content, err := os.ReadFile(filepath.Join(tmpDir, name))
		require.NoError(t, err, "Unexpected error reading test file")

		assert.NotEqual(
			t,
			strings.TrimSpace(expected),
			strings.TrimSpace(string(content)),
		)
	}

	require.Equal(t, 0, s.GetExitCode())
}

func Test_runner_run_listFiles(t *testing.T) {
	tmpDir := t.TempDir()

	updatedTestFiles := maps.Clone(testFiles)

	// File that doesn't need to be shortened
	updatedTestFiles["test3.go"] = "package main\n"

	runner := NewRunner()
	runner.listFiles = true
	runner.args = append(runner.args, writeTestFiles(t, updatedTestFiles, tmpDir)...)

	var buf bytes.Buffer

	s := newSequencer(1, &buf, os.Stderr)

	runner.run(s)

	require.Equal(t, 0, s.GetExitCode())

	// Only the first two files appear in the output list
	expectedPaths := []string{
		filepath.Join(tmpDir, "test1.go"),
		filepath.Join(tmpDir, "test2.go"),
	}

	actualPaths := strings.Split(strings.TrimSpace(buf.String()), "\n")

	slices.Sort(actualPaths)

	assert.Equal(
		t,
		expectedPaths,
		actualPaths,
	)
}

func writeTestFiles(
	t *testing.T,
	fileContents map[string]string,
	tmpDir string,
) []string {
	t.Helper()

	var filePaths []string

	for name, content := range fileContents {
		path := filepath.Join(tmpDir, name)

		filePaths = append(filePaths, path)

		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "Unexpected error-writing test file")
	}

	return filePaths
}
