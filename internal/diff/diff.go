package diff

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"golang.org/x/term"
)

const (
	ansiGreen = "\033[92m"
	ansiRed   = "\033[91m"
	ansiBlue  = "\033[94m"
	ansiEnd   = "\033[0m"
)

// Pretty prints colored, git-style diffs to the console.
func Pretty(path string, content, result []byte) (string, error) {
	if bytes.Equal(content, result) {
		return "", nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(content)),
		B:        difflib.SplitLines(string(result)),
		FromFile: path,
		ToFile:   path + ".shortened",
		Context:  3,
	}

	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return "", err
	}

	var builder strings.Builder

	for line := range strings.Lines(text) {
		line = strings.TrimRight(line, " ")
		switch {
		case !term.IsTerminal(int(os.Stdout.Fd())) && len(line) > 0:
			_, _ = fmt.Fprint(&builder, line)
		case strings.HasPrefix(line, "+"):
			_, _ = fmt.Fprint(&builder, ansiGreen, line, ansiEnd)
		case strings.HasPrefix(line, "-"):
			_, _ = fmt.Fprint(&builder, ansiRed, line, ansiEnd)
		case strings.HasPrefix(line, "^"):
			_, _ = fmt.Fprint(&builder, ansiBlue, line, ansiEnd)
		case len(line) > 0:
			_, _ = fmt.Fprintf(&builder, "%s", line)
		}
	}

	_, _ = fmt.Fprintln(&builder)

	return builder.String(), nil
}
