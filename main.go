package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/alecthomas/kingpin/v2"
	"github.com/golangci/golines/internal/diff"
	"github.com/golangci/golines/internal/formatter"
	"github.com/golangci/golines/shortener"
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

	// Arbitrarily limit in-flight work to 2MiB times the number of threads.
	//
	// The actual overhead for the parse tree and output will depend on the
	// specifics of the file, but this at least keeps the footprint of the process
	// roughly proportional to GOMAXPROCS.
	maxWeight := (2 << 20) * int64(runtime.GOMAXPROCS(0))

	s := newSequencer(maxWeight, os.Stdout, os.Stderr)

	run(s)

	os.Exit(s.GetExitCode())
}

func run(s *sequencer) {
	if deref(versionFlag) {
		fmt.Printf( //nolint:forbidigo
			"golines v%s\n\nbuild information:\n\tbuild date: %s\n\tgit commit ref: %s\n",
			version, date, commit,
		)

		return
	}

	if deref(profile) != "" {
		fdSem <- true

		f, err := os.Create(*profile)
		if err != nil {
			s.AddReport(fmt.Errorf("creating cpu profile: %w", err))
		}

		defer func() {
			_ = f.Close()

			<-fdSem
		}()

		_ = pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	NewRunner().run(s)
}

type Runner struct {
	args            []string
	ignoredDirs     []string
	ignoreGenerated bool
	dryRun          bool
	listFiles       bool
	writeOutput     bool

	shortener *shortener.Shortener

	extraFormatter *formatter.Executable
}

func NewRunner() *Runner {
	config := shortener.Config{
		MaxLen:          deref(maxLen),
		TabLen:          deref(tabLen),
		KeepAnnotations: deref(keepAnnotations),
		ShortenComments: deref(shortenComments),
		ReformatTags:    deref(reformatTags),
		DotFile:         deref(dotFile),
		ChainSplitDots:  deref(chainSplitDots),
		Logger:          slog.Default(),
	}

	return &Runner{
		args:            deref(paths),
		ignoredDirs:     deref(ignoredDirs),
		ignoreGenerated: deref(ignoreGenerated),
		dryRun:          deref(dryRun),
		listFiles:       deref(listFiles),
		writeOutput:     deref(writeOutput),

		shortener:      shortener.NewShortener(config),
		extraFormatter: formatter.NewExecutable(deref(baseFormatterCmd)),
	}
}

func (r *Runner) run(s *sequencer) {
	// Read input from stdin
	if len(r.args) == 0 {
		s.Add(0, func(rp *reporter) error {
			return r.processFile("<standard input>", nil, os.Stdin, rp)
		})

		return
	}

	// Read inputs from paths provided in arguments
	for _, arg := range r.args {
		switch info, err := os.Stat(arg); {
		case err != nil:
			s.AddReport(err)

		case !info.IsDir():
			if r.isIgnoredFile(arg) {
				return
			}

			s.Add(fileWeight(arg, info), func(rp *reporter) error {
				return r.processFile(arg, info, nil, rp)
			})

		default:
			// Path is a directory, walk it
			err = filepath.WalkDir(arg, func(path string, f fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if r.skipDir(path, f) {
					return filepath.SkipDir
				}

				if f.IsDir() {
					return nil
				}

				if r.isIgnoredFile(path) {
					return nil
				}

				info, err := f.Info()
				if err != nil {
					s.AddReport(err)

					return nil
				}

				s.Add(fileWeight(path, info), func(rp *reporter) error {
					return r.processFile(path, info, nil, rp)
				})

				return nil
			})
			if err != nil {
				s.AddReport(err)
			}
		}
	}
}

func (r *Runner) processFile(path string, info fs.FileInfo, in io.Reader, rp *reporter) error {
	slog.Debug("processing file", slog.String("path", path))

	content, err := readFile(path, info, in)
	if err != nil {
		return err
	}

	if r.ignoreGenerated && r.isGenerated(content) {
		return nil
	}

	// Do initial, non-line-length-aware formatting
	result, err := r.extraFormatter.Format(context.Background(), content)
	if err != nil {
		return err
	}

	result, err = r.shortener.Shorten(result)
	if err != nil {
		return err
	}

	// Do the final round of non-line-length-aware formatting after we've fixed up the comments
	result, err = r.extraFormatter.Format(context.Background(), result)
	if err != nil {
		return err
	}

	return r.handleOutput(path, content, result, info, rp)
}

// handleOutput generates output according to the value of the tool's
// flags; depending on the latter, the output might be written over
// the source file, printed to stdout, etc.
func (r *Runner) handleOutput(
	filename string,
	src, res []byte,
	info fs.FileInfo,
	rp *reporter,
) error {
	switch {
	case r.dryRun:
		pretty, err := diff.Pretty(filename, src, res)
		if err != nil {
			return err
		}

		if len(pretty) > 0 {
			_, _ = rp.Write([]byte(pretty))
		}

		return nil

	case r.listFiles:
		if !bytes.Equal(src, res) {
			_, _ = fmt.Fprintln(rp, filename)
		}

		return nil

	case r.writeOutput:
		if filename == "" {
			return errors.New("no path to write out to")
		}

		if bytes.Equal(src, res) {
			slog.Debug("content unchanged, skipping write")

			return nil
		}

		slog.Debug("content changed, writing output", slog.String("path", filename))

		perm := info.Mode().Perm()
		if err := writeFile(filename, src, res, perm, info.Size()); err != nil {
			return err
		}

		return nil

	default:
		_, _ = rp.Write(res)

		return nil
	}
}

func deref[T any](v *T) T { //nolint:ireturn
	if v == nil {
		var zero T

		return zero
	}

	return *v
}
