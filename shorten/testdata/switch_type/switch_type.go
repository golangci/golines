package testdata

import (
	"fmt"

	"github.com/dave/dst"
)

func _(node dst.Node) error {
	switch n := node.(type) {
	case dst.Decl:
		n.Decorations()

		return fmt.Errorf("This is a really long line that can be broken up twice %s %s", fmt.Sprintf("This is a really long sub-line that should be broken up more because %s %s", "xxxx", "yyyy"), fmt.Sprintf("A short one %d", 3))

	case dst.Expr:
		return fmt.Errorf("This is a really long line that can be broken up twice %s %s", fmt.Sprintf("This is a really long sub-line that should be broken up more because %s %s", "xxxx", "yyyy"), fmt.Sprintf("A short one %d", 3))

	case dst.Stmt:
		return fmt.Errorf("This is a really long line that can be broken up twice %s %s", fmt.Sprintf("This is a really long sub-line that should be broken up more because %s %s", "xxxx", "yyyy"), fmt.Sprintf("A short one %d", 3))

	default:
		return fmt.Errorf("This is a really long line that can be broken up twice %s %s", fmt.Sprintf("This is a really long sub-line that should be broken up more because %s %s", "xxxx", "yyyy"), fmt.Sprintf("A short one %d", 3))
	}
}
