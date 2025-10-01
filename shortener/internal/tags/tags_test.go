package tags

import (
	"testing"

	"github.com/dave/dst"
	"github.com/stretchr/testify/assert"
)

func TestHasMultipleTags(t *testing.T) {
	testCases := []struct {
		desc   string
		lines  []string
		assert assert.BoolAssertionFunc
	}{
		{
			desc:   "no tags",
			lines:  []string{"xxxxx"},
			assert: assert.False,
		},
		{
			desc:   "invalid tag",
			lines:  []string{"key   `xxxxx yyyy zzzz key:`"},
			assert: assert.False,
		},
		{
			desc:   "one key",
			lines:  []string{"key   `tagKey:\"tag value\"`"},
			assert: assert.False,
		},
		{
			desc:   "one key with whitespace",
			lines:  []string{"key   `  tagKey:\"tag value\"  `"},
			assert: assert.False,
		},
		{
			desc: "multiple keys",
			lines: []string{
				"xxxx",
				"key   `tagKey1:\"tag value1\"  tagKey2:\"tag value2\" `",
			},
			assert: assert.True,
		},
		{
			desc: "multiple keys with whitespace",
			lines: []string{
				"key   `  tagKey1:\"tag value1\" tagKey2:\"tag value2\"   tagKey3:\"tag value3\" `",
				"zzzz",
			},
			assert: assert.True,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			test.assert(t, HasMultipleTags(test.lines))
		})
	}
}

func TestFormatStructTags(t *testing.T) {
	testCases := []struct {
		desc     string
		list     []*dst.Field
		expected []string
	}{
		{
			desc: "align tags",
			list: []*dst.Field{
				{
					Names: []*dst.Ident{{Name: "key"}},
					Type:  &dst.Ident{Name: "string"},
					Tag: &dst.BasicLit{
						Value: "`tagKey1:\"tag value1\" tagKey2:\"tag value2\"   tagKey3:\"tag value3\" `",
					},
				},
				{
					Names: []*dst.Ident{{Name: "value"}},
					Type:  &dst.Ident{Name: "string"},
					Tag: &dst.BasicLit{
						Value: "`tagKey2:\"value2\" tagKey1:\"value1\"   tagKey3:\"value3\" `",
					},
				},
			},
			expected: []string{
				"`tagKey1:\"tag value1\" tagKey2:\"tag value2\" tagKey3:\"tag value3\"`",
				"`tagKey1:\"value1\"     tagKey2:\"value2\"     tagKey3:\"value3\"`",
			},
		},
		{
			desc: "no tags",
			list: []*dst.Field{
				{
					Names: []*dst.Ident{{Name: "key"}},
					Type:  &dst.Ident{Name: "string"},
				},
				{
					Names: []*dst.Ident{{Name: "value"}},
					Type:  &dst.Ident{Name: "string"},
				},
			},
		},
		{
			desc: "raw literal",
			list: []*dst.Field{
				{
					Names: []*dst.Ident{{Name: "key"}},
					Type:  &dst.Ident{Name: "string"},
					Tag: &dst.BasicLit{
						Value: "`  tagKey1:\"tag \\value1\"`",
					},
				},
				{
					Names: []*dst.Ident{{Name: "value"}},
					Type:  &dst.Ident{Name: "string"},
					Tag: &dst.BasicLit{
						Value: "`parameter:\"BAR\" delimiter:\"\\n\"`",
					},
				},
			},
			expected: []string{
				"`tagKey1:\"tag \\value1\"`",
				"`                      parameter:\"BAR\" delimiter:\"\\n\"`",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			fl := &dst.FieldList{
				List: test.list,
			}

			FormatStructTags(fl)

			var actual []string

			for _, field := range fl.List {
				if field == nil || field.Tag == nil {
					continue
				}

				actual = append(actual, field.Tag.Value)
			}

			assert.Equal(t, test.expected, actual)
		})
	}
}
