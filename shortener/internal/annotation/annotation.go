package annotation

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dave/dst"
)

const prefix = "// __golines:shorten:"

// Create generates the text of a comment that will annotate long lines.
func Create(length int) string {
	return fmt.Sprintf(
		"%s%d",
		prefix,
		length,
	)
}

// Is determines whether the given line is an annotation created with Create.
func Is(line string) bool {
	return strings.HasPrefix(
		strings.Trim(line, " \t"),
		prefix,
	)
}

// Has determines whether the given AST node has a line length annotation on it.
func Has(node dst.Node) bool {
	startDecorations := node.Decorations().Start.All()

	return len(startDecorations) > 0 &&
		Is(startDecorations[len(startDecorations)-1])
}

// HasTail determines whether the given AST node has a line length annotation at its end.
// This is needed to catch long function declarations with inline interface definitions.
func HasTail(node dst.Node) bool {
	endDecorations := node.Decorations().End.All()

	return len(endDecorations) > 0 &&
		Is(endDecorations[len(endDecorations)-1])
}

// HasRecursive determines whether the given node or one of its children has a
// golines annotation on it. It's currently implemented for function declarations, fields,
// call expressions, and selector expressions only.
func HasRecursive(node dst.Node) bool {
	if Has(node) {
		return true
	}

	switch n := node.(type) {
	case *dst.FuncDecl:
		if n.Type != nil && n.Type.Params != nil {
			for _, item := range n.Type.Params.List {
				if HasRecursive(item) {
					return true
				}
			}
		}
	case *dst.Field:
		return HasTail(n) || HasRecursive(n.Type)
	case *dst.SelectorExpr:
		return Has(n.Sel) || Has(n.X)
	case *dst.CallExpr:
		if HasRecursive(n.Fun) {
			return true
		}

		for _, arg := range n.Args {
			if Has(arg) {
				return true
			}
		}
	case *dst.InterfaceType:
		for _, field := range n.Methods.List {
			if HasRecursive(field) {
				return true
			}
		}
	}

	return false
}

// Parse returns the line length encoded in a golines annotation. If none is found,
// it returns -1.
func Parse(line string) int {
	if Is(line) {
		components := strings.SplitN(line, ":", 3)
		val, err := strconv.Atoi(components[2])
		if err != nil {
			return -1
		}
		return val
	}
	return -1
}
