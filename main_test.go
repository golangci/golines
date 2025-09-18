package main

import (
	"io"
	"os"
	"path/filepath"
	"sort"
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

	runner := NewRunner()
	runner.paths = []string{tmpDir}

	writeTestFiles(t, testFiles, tmpDir)

	err := runner.run()
	require.NoError(t, err)

	// Without writeOutput set to true, inputs should be unchanged
	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)

		bytes, err := os.ReadFile(path)
		require.NoError(t, err, "Unexpected error reading test file")

		assert.Equal(
			t,
			strings.TrimSpace(content),
			strings.TrimSpace(string(bytes)),
		)
	}

	runner.writeOutput = true

	err = runner.run()
	require.NoError(t, err)

	// Now, files should be modified in place
	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)

		bytes, err := os.ReadFile(path)
		require.NoError(t, err, "Unexpected error reading test file")

		assert.NotEqual(
			t,
			strings.TrimSpace(content),
			strings.TrimSpace(string(bytes)),
		)
	}
}

func Test_runner_run_filePaths(t *testing.T) {
	tmpDir := t.TempDir()

	runner := NewRunner()
	runner.writeOutput = true
	runner.paths = append(runner.paths, writeTestFiles(t, testFiles, tmpDir)...)

	err := runner.run()
	require.NoError(t, err)

	// Now, files should be modified in place
	for name, content := range testFiles {
		path := filepath.Join(tmpDir, name)

		bytes, err := os.ReadFile(path)
		require.NoError(t, err, "Unexpected error reading test file")

		assert.NotEqual(
			t,
			strings.TrimSpace(content),
			strings.TrimSpace(string(bytes)),
		)
	}
}

func Test_runner_run_listFiles(t *testing.T) {
	tmpDir := t.TempDir()

	runner := NewRunner()
	runner.listFiles = true

	updatedTestFiles := map[string]string{
		"test1.go": testFiles["test1.go"],
		"test2.go": testFiles["test2.go"],

		// File that doesn't need to be shortened
		"test3.go": "package main\n",
	}

	runner.paths = append(runner.paths, writeTestFiles(t, updatedTestFiles, tmpDir)...)

	output, err := captureStdout(t, runner)
	require.NoError(t, err)

	// Only first two files appear in output list
	expectedPaths := []string{
		filepath.Join(tmpDir, "test1.go"),
		filepath.Join(tmpDir, "test2.go"),
	}

	actualPaths := strings.Split(strings.TrimSpace(output), "\n")
	sort.Slice(actualPaths, func(i, j int) bool {
		return actualPaths[i] < actualPaths[j]
	})

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

	var tfp []string

	for name, content := range fileContents {
		path := filepath.Join(tmpDir, name)

		tfp = append(tfp, path)

		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err, "Unexpected error-writing test file")
	}

	return tfp
}

func captureStdout(t *testing.T, runner *Runner) (string, error) {
	t.Helper()

	origStdout := os.Stdout

	defer func() {
		os.Stdout = origStdout
	}()

	r, w, err := os.Pipe()
	require.NoError(t, err, "Unexpected error opening pipe")

	os.Stdout = w

	resultErr := runner.run()

	err = w.Close()
	require.NoError(t, err)

	outBytes, err := io.ReadAll(r)
	require.NoError(t, err, "Unexpected error reading result")

	return string(outBytes), resultErr
}
