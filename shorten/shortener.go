package shorten

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/golangci/golines/shorten/internal/annotation"
	"github.com/golangci/golines/shorten/internal/graph"
	"github.com/golangci/golines/shorten/internal/tags"
)

// Go directive (should be ignored).
// https://go.dev/doc/comment#syntax
var directivePattern = regexp.MustCompile(`\s*//(line |extern |export |[a-z0-9]+:[a-z0-9])`)

// The maximum number of shortening "rounds" that we'll allow.
// The shortening process should converge quickly,
// but we have this here as a safety mechanism to prevent loops that prevent termination.
const maxRounds = 20

// Config stores the configuration options exposed by a Shortener instance.
type Config struct {
	// MaxLen Max target width for each line
	MaxLen int

	// TabLen Width of a tab character
	TabLen int

	// KeepAnnotations Whether to keep annotations in the final result (for debugging only)
	KeepAnnotations bool

	// ShortenComments Whether to shorten comments
	ShortenComments bool

	// ReformatTags Whether to reformat struct tags in addition to shortening long lines
	ReformatTags bool

	// DotFile Path to write dot-formatted output to (for debugging only)
	DotFile string

	// ChainSplitDots Whether to split chain methods by putting dots at the ends of lines
	ChainSplitDots bool
}

// NewDefaultConfig returns a [Config] with default values.
func NewDefaultConfig() *Config {
	return &Config{
		MaxLen:          100,
		TabLen:          4,
		KeepAnnotations: false,
		ShortenComments: false,
		ReformatTags:    true,
		DotFile:         "",
		ChainSplitDots:  true,
	}
}

// Options is the type for configuring options of a [Shortener] instance.
type Options func(*Shortener)

// WithLogger sets the logger to use it for a [Shortener] instance.
func WithLogger(logger Logger) Options {
	return func(s *Shortener) {
		if logger != nil {
			s.logger = logger
		}
	}
}

// Shortener shortens a single go file according to a small set of user style preferences.
type Shortener struct {
	config *Config

	logger Logger
}

// NewShortener creates a new shortener instance from the provided config.
func NewShortener(config *Config, opts ...Options) *Shortener {
	if config == nil {
		config = NewDefaultConfig()
	}

	s := &Shortener{
		config: config,
		logger: &noopLogger{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Process shortens the provided golang file content bytes.
func (s *Shortener) Process(content []byte) ([]byte, error) {
	var round int

	var err error

	// Do initial, non-line-length-aware formatting
	content, err = format.Source(content)
	if err != nil {
		return nil, fmt.Errorf("error formatting source: %w", err)
	}

	for {
		s.logger.Debug("starting round", slog.Int("round", round))

		// Annotate all long lines
		lines := strings.Split(string(content), "\n")
		annotatedLines, linesToShorten := s.annotateLongLines(lines)

		var stop bool

		if linesToShorten == 0 {
			if round == 0 {
				if !s.config.ReformatTags || !tags.HasMultipleTags(lines) {
					stop = true
				}
			} else {
				stop = true
			}
		}

		if stop {
			s.logger.Debug("nothing more to shorten or reformat, stopping")

			break
		}

		content = []byte(strings.Join(annotatedLines, "\n"))

		// Generate AST
		result, err := decorator.Parse(content)
		if err != nil {
			return nil, err
		}

		if s.config.DotFile != "" {
			err = s.createDot(result)
			if err != nil {
				return nil, err
			}
		}

		// Process the file starting at the top-level declarations
		for _, decl := range result.Decls {
			s.formatNode(decl)
		}

		// Materialize output
		output := bytes.NewBuffer([]byte{})

		err = decorator.Fprint(output, result)
		if err != nil {
			return nil, fmt.Errorf("error parsing source: %w", err)
		}

		content = output.Bytes()

		round++

		if round > maxRounds {
			s.logger.Debug("hit max rounds, stopping")

			break
		}
	}

	if !s.config.KeepAnnotations {
		content = s.removeAnnotations(content)
	}

	if s.config.ShortenComments {
		content = s.shortenCommentsFunc(content)
	}

	// Do the final round of non-line-length-aware formatting after we've fixed up the comments
	content, err = format.Source(content)
	if err != nil {
		return nil, fmt.Errorf("error formatting source: %w", err)
	}

	return content, nil
}

// annotateLongLines adds specially formatted comments to all eligible lines that are longer than
// the configured target length. If a line already has one of these comments from a previous
// shortening round, then the comment contents are updated.
func (s *Shortener) annotateLongLines(lines []string) ([]string, int) {
	var annotatedLines []string

	linesToShorten := 0
	prevLen := -1

	for _, line := range lines {
		length := s.lineLen(line)

		if prevLen > -1 {
			if length <= s.config.MaxLen {
				// Shortening successful, remove previous annotation
				annotatedLines = annotatedLines[:len(annotatedLines)-1]
			} else if length < prevLen {
				// Replace annotation with a new length
				annotatedLines[len(annotatedLines)-1] = annotation.Create(length)
				linesToShorten++
			}
		} else if !isComment(line) && length > s.config.MaxLen {
			annotatedLines = append(
				annotatedLines,
				annotation.Create(length),
			)
			linesToShorten++
		}

		annotatedLines = append(annotatedLines, line)
		prevLen = annotation.Parse(line)
	}

	return annotatedLines, linesToShorten
}

// removeAnnotations removes all comments added by the annotateLongLines
// function above.
func (s *Shortener) removeAnnotations(content []byte) []byte {
	var cleanedLines []string

	lines := strings.SplitSeq(string(content), "\n")

	for line := range lines {
		if !annotation.Is(line) {
			cleanedLines = append(cleanedLines, line)
		}
	}

	return []byte(strings.Join(cleanedLines, "\n"))
}

// shortenCommentsFunc attempts to shorten long comments in the provided source. As noted
// in the repo README, this functionality has some quirks and is disabled by default.
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

// lineLen gets the width of the provided line after tab expansion.
func (s *Shortener) lineLen(line string) int {
	length := 0

	for _, char := range line {
		if char == '\t' {
			length += s.config.TabLen
		} else {
			length++
		}
	}

	return length
}

// formatNode formats the provided AST node. The appropriate helper function is called
// based on whether the node is a declaration, expression, statement, or spec.
func (s *Shortener) formatNode(node dst.Node) {
	switch n := node.(type) {
	case dst.Decl:
		s.logger.Debug("processing declaration", slog.Any("node", n))
		s.formatDecl(n)

	case dst.Expr:
		s.logger.Debug("processing expression", slog.Any("node", n))
		s.formatExpr(n, false, false)

	case dst.Stmt:
		s.logger.Debug("processing statement", slog.Any("node", n))
		s.formatStmt(n, false)

	case dst.Spec:
		s.logger.Debug("processing spec", slog.Any("node", n))
		s.formatSpec(n, false)

	default:
		s.logger.Debug(
			"got a node type that can't be shortened",
			slog.Any("node_type", reflect.TypeOf(n)),
		)
	}
}

// formatDecl formats an AST declaration node. These include function declarations,
// imports, and constants.
func (s *Shortener) formatDecl(decl dst.Decl) {
	switch d := decl.(type) {
	case *dst.FuncDecl:
		if d.Type != nil && d.Type.Params != nil && annotation.HasRecursive(d) {
			s.formatFieldList(d.Type.Params)
		}

		s.formatStmt(d.Body, false)

	case *dst.GenDecl:
		shouldShorten := annotation.Has(d)

		for _, spec := range d.Specs {
			s.formatSpec(spec, shouldShorten)
		}

	default:
		s.logger.Debug(
			"got a declaration type that can't be shortened",
			slog.Any("decl_type", reflect.TypeOf(d)),
		)
	}
}

// formatFieldList formats a field list in a function declaration.
func (s *Shortener) formatFieldList(fieldList *dst.FieldList) {
	for i, field := range fieldList.List {
		formatList(field, i)
	}
}

// formatStmt formats an AST statement node. Among other examples, these include assignments,
// case clauses, for statements, if statements, and select statements.
func (s *Shortener) formatStmt(stmt dst.Stmt, force bool) {
	stmtType := reflect.TypeOf(stmt)

	// Explicitly check for nil statements
	if reflect.ValueOf(stmt) == reflect.Zero(stmtType) {
		return
	}

	shouldShorten := force || annotation.Has(stmt)

	switch st := stmt.(type) {
	case *dst.AssignStmt:
		for _, expr := range st.Rhs {
			s.formatExpr(expr, shouldShorten, false)
		}

	case *dst.BlockStmt:
		for _, stmt := range st.List {
			s.formatStmt(stmt, false)
		}

	case *dst.CaseClause:
		if shouldShorten {
			for _, arg := range st.List {
				arg.Decorations().After = dst.NewLine

				s.formatExpr(arg, false, false)
			}
		}

		for _, stmt := range st.Body {
			s.formatStmt(stmt, false)
		}

	case *dst.CommClause:
		for _, stmt := range st.Body {
			s.formatStmt(stmt, false)
		}

	case *dst.DeclStmt:
		s.formatDecl(st.Decl)

	case *dst.DeferStmt:
		s.formatExpr(st.Call, shouldShorten, false)

	case *dst.ExprStmt:
		s.formatExpr(st.X, shouldShorten, false)

	case *dst.ForStmt:
		s.formatStmt(st.Body, false)

	case *dst.GoStmt:
		s.formatExpr(st.Call, shouldShorten, false)

	case *dst.IfStmt:
		s.formatExpr(st.Cond, shouldShorten, false)
		s.formatStmt(st.Body, false)

		if st.Init != nil {
			s.formatStmt(st.Init, shouldShorten)
		}

	case *dst.RangeStmt:
		s.formatStmt(st.Body, false)

	case *dst.ReturnStmt:
		for _, expr := range st.Results {
			s.formatExpr(expr, shouldShorten, false)
		}

	case *dst.SelectStmt:
		s.formatStmt(st.Body, false)

	case *dst.SwitchStmt:
		s.formatStmt(st.Body, false)

	default:
		if shouldShorten {
			s.logger.Debug(
				"got a statement type that can't be shortened",
				slog.Any("stmt_type", stmtType),
			)
		}
	}
}

// formatExpr formats an AST expression node. These include uniary and binary expressions, function
// literals, and key/value pair statements, among others.
func (s *Shortener) formatExpr(expr dst.Expr, force, isChain bool) {
	shouldShorten := force || annotation.Has(expr)

	switch e := expr.(type) {
	case *dst.BinaryExpr:
		if (e.Op == token.LAND || e.Op == token.LOR) && shouldShorten {
			if e.Y.Decorations().Before == dst.NewLine {
				s.formatExpr(e.X, force, isChain)
			} else {
				e.Y.Decorations().Before = dst.NewLine
			}
		} else {
			s.formatExpr(e.X, shouldShorten, isChain)
			s.formatExpr(e.Y, shouldShorten, isChain)
		}

	case *dst.CallExpr:
		shortenChildArgs := shouldShorten || annotation.HasRecursive(e)

		_, ok := e.Fun.(*dst.SelectorExpr)

		if ok && shortenChildArgs &&
			s.config.ChainSplitDots && (isChain || chainLength(e) > 1) {
			e.Decorations().After = dst.NewLine

			for _, arg := range e.Args {
				s.formatExpr(arg, false, true)
			}

			s.formatExpr(e.Fun, shouldShorten, true)
		} else {
			for i, arg := range e.Args {
				if shortenChildArgs {
					formatList(arg, i)
				}

				s.formatExpr(arg, false, isChain)
			}

			s.formatExpr(e.Fun, shouldShorten, isChain)
		}

	case *dst.CompositeLit:
		if shouldShorten {
			for i, element := range e.Elts {
				if i == 0 {
					element.Decorations().Before = dst.NewLine
				}

				element.Decorations().After = dst.NewLine
			}
		}

		for _, element := range e.Elts {
			s.formatExpr(element, false, isChain)
		}

	case *dst.FuncLit:
		s.formatStmt(e.Body, false)

	case *dst.FuncType:
		if shouldShorten {
			s.formatFieldList(e.Params)
		}

	case *dst.InterfaceType:
		for _, method := range e.Methods.List {
			if annotation.Has(method) {
				s.formatExpr(method.Type, true, isChain)
			}
		}

	case *dst.KeyValueExpr:
		s.formatExpr(e.Value, shouldShorten, isChain)

	case *dst.SelectorExpr:
		s.formatExpr(e.X, shouldShorten, isChain)

	case *dst.StructType:
		if s.config.ReformatTags {
			tags.FormatStructTags(e.Fields)
		}

	case *dst.UnaryExpr:
		s.formatExpr(e.X, shouldShorten, isChain)

	default:
		if shouldShorten {
			s.logger.Debug(
				"got an expression type that can't be shortened",
				slog.Any("expr_type", reflect.TypeOf(e)),
			)
		}
	}
}

// formatSpec formats an AST spec node. These include type specifications, among other things.
func (s *Shortener) formatSpec(spec dst.Spec, force bool) {
	shouldShorten := annotation.Has(spec) || force

	switch sp := spec.(type) {
	case *dst.ValueSpec:
		for _, expr := range sp.Values {
			s.formatExpr(expr, shouldShorten, false)
		}

	case *dst.TypeSpec:
		s.formatExpr(sp.Type, false, false)

	default:
		if shouldShorten {
			s.logger.Debug(
				"got a spec type that can't be shortened",
				slog.Any("spec_type", reflect.TypeOf(sp)),
			)
		}
	}
}

func (s *Shortener) createDot(result dst.Node) error {
	dotFile, err := os.Create(s.config.DotFile)
	if err != nil {
		return err
	}

	defer dotFile.Close()

	s.logger.Debug("writing dot file output", slog.String("file", s.config.DotFile))

	return graph.CreateDot(result, dotFile)
}

func formatList(node dst.Node, index int) {
	decorations := node.Decorations()

	if index == 0 {
		decorations.Before = dst.NewLine
	} else {
		decorations.Before = dst.None
	}

	decorations.After = dst.NewLine
}

// chainLength determines the length of the function call chain in an expression.
func chainLength(callExpr *dst.CallExpr) int {
	numCalls := 1
	currCall := callExpr

	for {
		selectorExpr, ok := currCall.Fun.(*dst.SelectorExpr)
		if !ok {
			break
		}

		currCall, ok = selectorExpr.X.(*dst.CallExpr)
		if !ok {
			break
		}

		numCalls++
	}

	return numCalls
}

// isComment determines whether the provided line is a non-block comment.
func isComment(line string) bool {
	return strings.HasPrefix(strings.Trim(line, " \t"), "//")
}

// isDirective determines whether the provided line is a directive, e.g., for `go:generate`.
func isDirective(line string) bool {
	return directivePattern.MatchString(line)
}
