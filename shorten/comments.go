package shorten

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/golangci/golines/shorten/internal/annotation"
)

// Go directive (should be ignored).
// https://go.dev/doc/comment#syntax
var directivePattern = regexp.MustCompile(`\s*//(line |extern |export |[a-z0-9]+:[a-z0-9])`)

// shortenCommentsFunc attempts to shorten long comments in the provided source.
//
// As noted in the repo README,
// this functionality has some quirks and is disabled by default.
func (s *Shortener) shortenCommentsFunc(content []byte) []byte {
	var cleanedLines []string

	var words []string // all words in a contiguous sequence of long comments

	prefix := ""

	lines := strings.SplitSeq(string(content), "\n")
	for line := range lines {
		if isComment(line) && !annotation.Is(line) &&
			!isDirective(line) &&
			s.lineLen(line) > s.config.MaxLen {
			start := strings.Index(line, "//")
			prefix = line[0:(start + 2)]
			trimmedLine := strings.Trim(line[(start+2):], " ")
			currLineWords := strings.Split(trimmedLine, " ")
			words = append(words, currLineWords...)
		} else {
			// Reflow the accumulated `words` before appending the unprocessed `line`.
			currLineLen := 0

			var currLineWords []string

			maxCommentLen := s.config.MaxLen - s.lineLen(prefix)
			for _, word := range words {
				if currLineLen > 0 && currLineLen+1+len(word) > maxCommentLen {
					cleanedLines = append(
						cleanedLines,
						fmt.Sprintf(
							"%s %s",
							prefix,
							strings.Join(currLineWords, " "),
						),
					)
					currLineWords = []string{}
					currLineLen = 0
				}

				currLineWords = append(currLineWords, word)
				currLineLen += 1 + len(word)
			}

			if currLineLen > 0 {
				cleanedLines = append(
					cleanedLines,
					fmt.Sprintf(
						"%s %s",
						prefix,
						strings.Join(currLineWords, " "),
					),
				)
			}

			words = []string{}

			cleanedLines = append(cleanedLines, line)
		}
	}

	return []byte(strings.Join(cleanedLines, "\n"))
}

// isDirective determines whether the provided line is a directive, e.g., for `go:generate`.
func isDirective(line string) bool {
	return directivePattern.MatchString(line)
}

// isComment determines whether the provided line is a non-block comment.
func isComment(line string) bool {
	return strings.HasPrefix(strings.Trim(line, " \t"), "//")
}
