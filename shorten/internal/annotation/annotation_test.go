package annotation

import (
	"testing"

	"github.com/dave/dst"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	assert.Equal(t, "//golines:shorten:5", Create(5))
}

func TestIs(t *testing.T) {
	testCases := []struct {
		desc   string
		line   string
		assert assert.BoolAssertionFunc
	}{
		{
			desc:   "annotation",
			line:   "//golines:shorten:5",
			assert: assert.True,
		},
		{
			desc:   "not an annotation",
			line:   "// not an annotation",
			assert: assert.False,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			test.assert(t, Is(test.line))
		})
	}
}

func TestHas(t *testing.T) {
	testCases := []struct {
		desc   string
		node   *dst.Ident
		assert assert.BoolAssertionFunc
	}{
		{
			desc: "",
			node: &dst.Ident{
				Name: "x",
				Decs: dst.IdentDecorations{
					NodeDecs: dst.NodeDecs{
						Start: []string{
							"// not an annotation",
							Create(55),
						},
					},
				},
			},
			assert: assert.True,
		},
		{
			desc: "",
			node: &dst.Ident{
				Name: "x",
				Decs: dst.IdentDecorations{
					NodeDecs: dst.NodeDecs{
						Start: []string{
							"// not an annotation",
						},
					},
				},
			},
			assert: assert.False,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			test.assert(t, Has(test.node))
		})
	}
}

func TestHasTail(t *testing.T) {
	testCases := []struct {
		desc   string
		node   *dst.Ident
		assert assert.BoolAssertionFunc
	}{
		{
			desc: "tail",
			node: &dst.Ident{
				Name: "x",
				Decs: dst.IdentDecorations{
					NodeDecs: dst.NodeDecs{
						End: []string{
							"// not an annotation",
							Create(55),
						},
					},
				},
			},
			assert: assert.True,
		},
		{
			desc: "no tail",
			node: &dst.Ident{
				Name: "x",
				Decs: dst.IdentDecorations{
					NodeDecs: dst.NodeDecs{
						End: []string{
							"// not an annotation",
						},
					},
				},
			},
			assert: assert.False,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			test.assert(t, HasTail(test.node))
		})
	}
}

func TestParse(t *testing.T) {
	testCases := []struct {
		desc     string
		line     string
		expected int
	}{
		{
			desc:     "valid annotation",
			line:     "//golines:shorten:5",
			expected: 5,
		},
		{
			desc:     "not a number",
			line:     "//golines:shorten:not_a_number",
			expected: -1,
		},
		{
			desc:     "not an annotation",
			line:     "// not an annotation",
			expected: -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, test.expected, Parse(test.line))
		})
	}
}
