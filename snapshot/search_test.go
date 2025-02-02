package snapshot

import (
	"testing"

	"github.com/PlakarKorp/plakar/search"
	"github.com/stretchr/testify/require"
)

func TestSearch(t *testing.T) {
	snap := generateSnapshot(t, nil)
	defer snap.Close()

	err := snap.repository.RebuildState()
	require.NoError(t, err)

	type testCase struct {
		name     string
		query    string
		expected int
	}

	testCases := []testCase{
		{
			name:     "filename=dummy.txt",
			query:    "filename=dummy.txt",
			expected: 1,
		},
		{
			name:     `filename="unknown.txt"`,
			query:    `filename="unknown.txt"`,
			expected: 0,
		},
		{
			name:     "filename!=nonexistent.txt",
			query:    "filename!=nonexistent.txt",
			expected: 1,
		},
		{
			name:     "filename<=a",
			query:    "filename<=a",
			expected: 0,
		},
		{
			name:     "filename<a",
			query:    "filename<a",
			expected: 0,
		},
		{
			name:     "filename>=a",
			query:    "filename>=a",
			expected: 1,
		},
		{
			name:     "filename>a",
			query:    "filename>a",
			expected: 1,
		},
		{
			name:     "filename~=dummy",
			query:    "filename~=dummy",
			expected: 1,
		},
		{
			name:     `contenttype="text/plain; charset=utf-8"`,
			query:    `contenttype="text/plain; charset=utf-8"`,
			expected: 1,
		},
		{
			name:     `contenttype!="text/html; charset=utf-8"`,
			query:    `contenttype!="text/html; charset=utf-8"`,
			expected: 1,
		},
		{
			name:     `contenttype~="text/plain"`,
			query:    `contenttype~="text/plain"`,
			expected: 1,
		},
		{
			name:     `contenttype<="text/zzz"`,
			query:    `contenttype<="text/zzz"`,
			expected: 1,
		},
		{
			name:     `contenttype<"text/zzz"`,
			query:    `contenttype<"text/zzz"`,
			expected: 1,
		},
		{
			name:     `contenttype>="text/aaa"`,
			query:    `contenttype>="text/aaa"`,
			expected: 1,
		},
		{
			name:     `contenttype>"text/aaa"`,
			query:    `contenttype>"text/aaa"`,
			expected: 1,
		},
		{
			name:     `size="5"`,
			query:    `size="5"`,
			expected: 1,
		},
		{
			name:     `size!=10`,
			query:    `size!=10`,
			expected: 1,
		},
		{
			name:     `size<20`,
			query:    `size<20`,
			expected: 1,
		},
		{
			name:     `size<=20`,
			query:    `size<=20`,
			expected: 1,
		},
		{
			name:     `size>0`,
			query:    `size>0`,
			expected: 1,
		},
		{
			name:     `size>=0`,
			query:    `size>=0`,
			expected: 1,
		},
		{
			name:     `size=wrongformat`,
			query:    `size=wrongformat`,
			expected: 0,
		},
		{
			name:     `entropy=1.9219280948873625`,
			query:    `entropy=1.9219280948873625`,
			expected: 1,
		},
		{
			name:     `entropy!=2`,
			query:    `entropy!=2`,
			expected: 1,
		},
		{
			name:     `entropy<2`,
			query:    `entropy<2`,
			expected: 1,
		},
		{
			name:     `entropy<=2`,
			query:    `entropy<=2`,
			expected: 1,
		},
		{
			name:     `entropy>1`,
			query:    `entropy>1`,
			expected: 1,
		},
		{
			name:     `entropy>=1`,
			query:    `entropy>=1`,
			expected: 1,
		},
		{
			name:     `filename!=wrong AND entropy>=1`,
			query:    `filename!=wrong AND entropy>=1`,
			expected: 1,
		},
		{
			name:     `filename!=wrong AND unknown>=1`,
			query:    `filename!=wrong AND unknown>=1`,
			expected: 0,
		},
		{
			name:     `filename!=wrong OR entropy>=1`,
			query:    `filename!=wrong OR entropy>=1`,
			expected: 1,
		},
		{
			name:     `filename!=wrong NOT entropy>=1`,
			query:    `filename!=wrong NOT entropy>=1`,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := snap.Search("/tmp/", tc.query)
			require.NoError(t, err)

			items := make([]search.FileEntry, 0)
			for result := range results {
				if entry, isFilename := result.(search.FileEntry); isFilename {
					items = append(items, entry)
				}
			}
			require.Equal(t, tc.expected, len(items))
		})
	}
}
