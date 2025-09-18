package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"slices"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/segmentio/golines/internal/diff"
	"github.com/segmentio/golines/shortener"
)

// these values are provided automatically by Goreleaser.
// ref: https://goreleaser.com/customization/builds/
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	// Flags.
	baseFormatterCmd = kingpin.Flag(
		"base-formatter",
		"Base formatter to use").Default("").String()
	chainSplitDots = kingpin.Flag(
		"chain-split-dots",
		"Split chained methods on the dots as opposed to the arguments").
		Default("true").Bool()
	debug = kingpin.Flag(
		"debug",
		"Show debug output").Short('d').Default("false").Bool()
	dotFile = kingpin.Flag(
		"dot-file",
		"Path to dot representation of the AST graph").Default("").String()
	dryRun = kingpin.Flag(
		"dry-run",
		"Show diffs without writing anything").Default("false").Bool()
	ignoreGenerated = kingpin.Flag(
		"ignore-generated",
		"Ignore generated go files").Default("true").Bool()
	ignoredDirs = kingpin.Flag(
		"ignored-dirs",
		"Directories to ignore").Default("vendor", "testdata", "node_modules").Strings()
	keepAnnotations = kingpin.Flag(
		"keep-annotations",
		"Keep shortening annotations in the final output").Default("false").Bool()
	listFiles = kingpin.Flag(
		"list-files",
		"List files that would be reformatted by this tool").Short('l').Default("false").Bool()
	maxLen = kingpin.Flag(
		"max-len",
		"Target maximum line length").Short('m').Default("100").Int()
	profile = kingpin.Flag(
		"profile",
		"Path to profile output").Default("").String()
	reformatTags = kingpin.Flag(
		"reformat-tags",
		"Reformat struct tags").Default("true").Bool()
	shortenComments = kingpin.Flag(
		"shorten-comments",
		"Shorten single-line comments").Default("false").Bool()
	tabLen = kingpin.Flag(
		"tab-len",
		"Length of a tab").Short('t').Default("4").Int()
	versionFlag = kingpin.Flag(
		"version",
		"Print out version and exit").Default("false").Bool()
	writeOutput = kingpin.Flag(
		"write-output",
		"Write output to source instead of stdout").Short('w').Default("false").Bool()

	// Args.
	paths = kingpin.Arg(
		"paths",
		"Paths to format",
	).Strings()
)

func main() {
	kingpin.Parse()

	if deref(debug) {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	err := run()
	if err != nil {
		slog.Error("run", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	if deref(versionFlag) {
		fmt.Printf("golines v%s\n\nbuild information:\n\tbuild date: %s\n\tgit commit ref: %s\n",
			version, date, commit)

		return nil
	}

	if deref(profile) != "" {
		f, err := os.Create(*profile)
		if err != nil {
			return fmt.Errorf("create profile: %w", err)
		}

		_ = pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	return NewRunner().run()
}

type Runner struct {
	paths           []string
	ignoredDirs     []string
	ignoreGenerated bool
	dryRun          bool
	listFiles       bool
	writeOutput     bool

	shortener *shortener.Shortener
}

func NewRunner() *Runner {
	return &Runner{
		paths:           deref(paths),
		ignoredDirs:     deref(ignoredDirs),
		ignoreGenerated: deref(ignoreGenerated),
		dryRun:          deref(dryRun),
		listFiles:       deref(listFiles),
		writeOutput:     deref(writeOutput),

		shortener: shortener.NewShortener(shortener.Config{
			MaxLen:           deref(maxLen),
			TabLen:           deref(tabLen),
			KeepAnnotations:  deref(keepAnnotations),
			ShortenComments:  deref(shortenComments),
			ReformatTags:     deref(reformatTags),
			IgnoreGenerated:  deref(ignoreGenerated),
			DotFile:          deref(dotFile),
			BaseFormatterCmd: deref(baseFormatterCmd),
			ChainSplitDots:   deref(chainSplitDots),
			Logger:           slog.Default(),
		}),
	}
}

func (r *Runner) run() error {
	// Read input from stdin
	if len(r.paths) == 0 {
		content, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		result, err := r.shortener.Shorten(content)
		if err != nil {
			return err
		}

		err = r.handleOutput("", content, result)
		if err != nil {
			return err
		}

		return nil
	}

	// Read inputs from paths provided in arguments
	for _, path := range r.paths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		// Path is a file
		if !info.IsDir() {
			if r.isIgnoredFile(path) {
				return nil
			}

			err = r.process(path)
			if err != nil {
				return err
			}

			continue
		}

		// Path is a directory, walk it
		err = filepath.Walk(path, func(subPath string, f os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if r.skipDir(subPath, f) {
				return filepath.SkipDir
			}

			if f.IsDir() {
				return nil
			}

			if r.isIgnoredFile(subPath) {
				return nil
			}

			return r.process(subPath)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) process(path string) error {
	slog.Debug("processing file", slog.String("path", path))

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result, err := r.shortener.Shorten(content)
	if err != nil {
		return err
	}

	return r.handleOutput(path, content, result)
}

// handleOutput generates output according to the value of the tool's
// flags; depending on the latter, the output might be written over
// the source file, printed to stdout, etc.
func (r *Runner) handleOutput(path string, content, result []byte) error {
	switch {
	case r.dryRun:
		pretty, err := diff.Pretty(path, content, result)
		if err != nil {
			return err
		}

		if len(pretty) > 0 {
			fmt.Println(pretty)
		}

		return nil

	case r.listFiles:
		if !bytes.Equal(content, result) {
			fmt.Println(path)
		}

		return nil

	case r.writeOutput:
		if path == "" {
			return errors.New("no path to write out to")
		}

		if bytes.Equal(content, result) {
			slog.Debug("content unchanged, skipping write")

			return nil
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		slog.Debug("content changed, writing output", slog.String("path", path))

		return os.WriteFile(path, result, info.Mode())

	default:
		fmt.Print(string(result))

		return nil
	}
}

func (r *Runner) skipDir(subPath string, f os.FileInfo) bool {
	if f.IsDir() {
		switch f.Name() {
		case "vendor", "testdata", "node_modules":
			return true

		default:
			return f.Name() != "." && strings.HasPrefix(f.Name(), ".")
		}
	}

	if len(r.ignoredDirs) == 0 {
		return false
	}

	parts := strings.Split(subPath, "/")
	for _, part := range parts {
		if slices.Contains(r.ignoredDirs, part) {
			return true
		}
	}

	return false
}

func (r *Runner) isIgnoredFile(path string) bool {
	if !strings.HasSuffix(path, ".go") {
		return true
	}

	_, fileName := filepath.Split(path)

	return r.ignoreGenerated && strings.HasPrefix(fileName, "generated_")
}

func deref[T any](v *T) T { //nolint:ireturn
	if v == nil {
		var zero T

		return zero
	}

	return *v
}
