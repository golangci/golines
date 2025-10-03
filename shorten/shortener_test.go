package shorten

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testdataDir = "testdata"

// TestShortener verifies the core shortening functionality on the files in the `testdata` directory.
// To update the expected outputs, run tests with the `REGENERATE_TEST_OUTPUTS` environment variable set to `true`.
func TestShortener(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	}))

	for file, config := range loadTestCases(t) {
		t.Run(file, func(t *testing.T) {
			t.Parallel()

			shortener := NewShortener(config, WithLogger(logger))

			content, err := os.ReadFile(file)
			require.NoErrorf(t, err,
				"Unexpected error reading fixture %s",
				file,
			)

			shortenedContent, err := shortener.Process(content)
			require.NoError(t, err)

			expectedPath := strings.TrimSuffix(file, filepath.Ext(file)) + ".go.golden"

			if os.Getenv("REGENERATE_TEST_OUTPUTS") == "true" {
				err := os.WriteFile(expectedPath, shortenedContent, 0o644)
				require.NoErrorf(t, err,
					"Unexpected error writing output file %s",
					expectedPath,
				)
			}

			expectedContent, err := os.ReadFile(expectedPath)
			require.NoErrorf(t, err,
				"Unexpected error reading expected file %s",
				expectedPath,
			)

			assert.Equal(t, string(expectedContent), string(shortenedContent))

			if config != nil && config.DotFile != "" {
				expectedDotFile := strings.TrimSuffix(file, filepath.Ext(file)) + ".dot"

				dotFileContent, err := os.ReadFile(config.DotFile)
				require.NoError(t, err)

				if os.Getenv("REGENERATE_TEST_OUTPUTS") == "true" {
					err = os.WriteFile(expectedDotFile, dotFileContent, 0o644)
					require.NoErrorf(t, err,
						"Unexpected error writing output file %s",
						expectedDotFile,
					)
				}

				expectedDotContent, err := os.ReadFile(expectedDotFile)
				require.NoError(t, err)

				assert.Equal(t, string(expectedDotContent), string(dotFileContent))
			}
		})
	}
}

func loadTestCases(t *testing.T) map[string]*Config {
	t.Helper()

	dotDir := t.TempDir()

	testCases := map[string]*Config{}

	err := filepath.WalkDir(testdataDir, func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}

		fileWithoutExt := strings.TrimSuffix(file, filepath.Ext(file))

		cfgFile := fileWithoutExt + ".json"

		_, err = os.Stat(cfgFile)
		if errors.Is(err, fs.ErrNotExist) {
			testCases[file] = NewDefaultConfig()

			return nil
		}

		tmpl, err := template.New(filepath.Base(cfgFile)).ParseFiles(cfgFile)
		require.NoError(t, err, cfgFile)

		buf := &bytes.Buffer{}

		dotFile := filepath.Join(dotDir, filepath.Base(fileWithoutExt)+".dot")

		data := map[string]string{
			"dotFile": jsonEncoded(t, dotFile),
		}

		err = tmpl.Execute(buf, data)
		require.NoError(t, err, cfgFile)

		cfg := &Config{}

		err = json.Unmarshal(buf.Bytes(), cfg) //nolint:musttag
		require.NoError(t, err, cfgFile)

		testCases[file] = cfg

		return nil
	})
	require.NoError(t, err)

	return testCases
}

// jsonEncoded escapes a string because of the file path separator on Windows.
func jsonEncoded(t *testing.T, dotFile string) string {
	t.Helper()

	escaped, err := json.Marshal(dotFile)
	require.NoError(t, err)

	return string(escaped)[1 : len(escaped)-1]
}
