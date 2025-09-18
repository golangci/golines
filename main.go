package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/segmentio/golines/internal/diff"
	"github.com/segmentio/golines/shortener"
)

var (
	// these values are provided automatically by Goreleaser
	//   ref: https://goreleaser.com/customization/builds/
	version = "dev"
	commit  = "none"
	date    = "unknown"

	// Flags
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
		"Path to dot representation of AST graph").Default("").String()
	dryRun = kingpin.Flag(
		"dry-run",
		"Show diffs without writing anything").Default("false").Bool()
	ignoreGenerated = kingpin.Flag(
		"ignore-generated",
		"Ignore generated go files").Default("true").Bool()
	ignoredDirs = kingpin.Flag(
		"ignored-dirs",
		"Directories to ignore").Default("").Strings()
	keepAnnotations = kingpin.Flag(
		"keep-annotations",
		"Keep shortening annotations in final output").Default("false").Bool()
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

	// Args
	paths = kingpin.Arg(
		"paths",
		"Paths to format",
	).Strings()
)

func main() {
	kingpin.Parse()
	if *debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if *versionFlag {
		fmt.Printf("golines v%s\n\nbuild information:\n\tbuild date: %s\n\tgit commit ref: %s\n",
			version, date, commit)
		return
	}

	if *profile != "" {
		f, err := os.Create(*profile)
		if err != nil {
			slog.Error("create profile", slog.Any("error", err))
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	err := run()
	if err != nil {
		slog.Error("run", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	config := shortener.Config{
		MaxLen:           *maxLen,
		TabLen:           *tabLen,
		KeepAnnotations:  *keepAnnotations,
		ShortenComments:  *shortenComments,
		ReformatTags:     *reformatTags,
		IgnoreGenerated:  *ignoreGenerated,
		DotFile:          *dotFile,
		BaseFormatterCmd: *baseFormatterCmd,
		ChainSplitDots:   *chainSplitDots,
		Logger:           slog.Default(),
	}
	shortener := shortener.NewShortener(config)

	if len(*paths) == 0 {
		// Read input from stdin
		contents, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		result, err := shortener.Shorten(contents)
		if err != nil {
			return err
		}
		err = handleOutput("", contents, result)
		if err != nil {
			return err
		}
	} else {
		// Read inputs from paths provided in arguments
		for _, path := range *paths {
			switch info, err := os.Stat(path); {
			case err != nil:
				return err
			case info.IsDir():
				// Path is a directory- walk it
				err = filepath.Walk(
					path,
					func(subPath string, f os.FileInfo, err error) error {
						if err != nil {
							return err
						}

						if f.IsDir() && skipDir(f.Name()) {
							return fs.SkipDir
						}

						components := strings.Split(subPath, "/")
						for _, component := range components {
							for _, ignoredDir := range *ignoredDirs {
								if component == ignoredDir {
									return filepath.SkipDir
								}
							}
						}

						if !f.IsDir() && strings.HasSuffix(subPath, ".go") {
							// Shorten file and generate output
							contents, result, err := processFile(shortener, subPath)
							if err != nil {
								return err
							}
							err = handleOutput(subPath, contents, result)
							if err != nil {
								return err
							}
						}

						return nil
					},
				)
				if err != nil {
					return err
				}
			default:
				// Path is a file
				contents, result, err := processFile(shortener, path)
				if err != nil {
					return err
				}
				err = handleOutput(path, contents, result)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// processFile uses the provided Shortener instance to shorten the lines
// in a file. It returns the original contents (useful for debugging), the
// shortened version, and an error.
func processFile(shortener *shortener.Shortener, path string) ([]byte, []byte, error) {
	_, fileName := filepath.Split(path)
	if *ignoreGenerated && strings.HasPrefix(fileName, "generated_") {
		return nil, nil, nil
	}

	slog.Debug("processing file", slog.String("path", path))

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	result, err := shortener.Shorten(contents)
	return contents, result, err
}

// handleOutput generates output according to the value of the tool's
// flags; depending on the latter, the output might be written over
// the source file, printed to stdout, etc.
func handleOutput(path string, contents []byte, result []byte) error {
	if contents == nil {
		return nil
	} else if *dryRun {
		return diff.Pretty(path, contents, result)
	} else if *listFiles {
		if !bytes.Equal(contents, result) {
			fmt.Println(path)
		}

		return nil
	} else if *writeOutput {
		if path == "" {
			return errors.New("no path to write out to")
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if bytes.Equal(contents, result) {
			slog.Debug("contents unchanged, skipping write")
			return nil
		}

		slog.Debug("contents changed, writing output", slog.String("path", path))
		return os.WriteFile(path, result, info.Mode())
	}

	fmt.Print(string(result))
	return nil

}

func skipDir(name string) bool {
	switch name {
	case "vendor", "testdata", "node_modules":
		return true

	default:
		return strings.HasPrefix(name, ".")
	}
}
