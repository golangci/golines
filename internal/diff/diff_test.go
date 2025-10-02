package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPretty(t *testing.T) {
	testCases := []struct {
		desc     string
		content  string
		result   string
		expected string
	}{
		{
			desc:    "simple diff",
			content: "line 1\nline 2\n",
			result:  "line 1\nline 2 modified\n",
			expected: `diff example.txt example.txt.shortened
--- example.txt
+++ example.txt.shortened
@@ -1,2 +1,2 @@
 line 1
-line 2
+line 2 modified
`,
		},
		{
			desc:     "no diff",
			content:  "line 1\nline 2",
			result:   "line 1\nline 2",
			expected: "",
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			output := Pretty("example.txt", []byte(test.content), []byte(test.result))

			assert.Equal(t, test.expected, string(output))
		})
	}
}
