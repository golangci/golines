package annotation

import (
	"testing"

	"github.com/dave/dst"
	"github.com/stretchr/testify/assert"
)

func Test_strings(t *testing.T) {
	assert.Equal(t, "// __golines:shorten:5", Create(5))
	assert.Equal(t, 5, Parse("// __golines:shorten:5"))
	assert.Equal(t, -1, Parse("// __golines:shorten:not_a_number"))
	assert.Equal(t, -1, Parse("// not an annotation"))
	assert.True(t, Is("// __golines:shorten:5"))
	assert.False(t, Is("// not an annotation"))
}

func TestHas(t *testing.T) {
	node1 := &dst.Ident{
		Name: "x",
		Decs: dst.IdentDecorations{
			NodeDecs: dst.NodeDecs{
				Start: []string{
					"// not an annotation",
					Create(55),
				},
			},
		},
	}
	assert.True(t, Has(node1))

	node2 := &dst.Ident{
		Name: "x",
		Decs: dst.IdentDecorations{
			NodeDecs: dst.NodeDecs{
				Start: []string{
					"// not an annotation",
				},
			},
		},
	}
	assert.False(t, Has(node2))
}
