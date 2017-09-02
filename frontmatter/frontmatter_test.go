package frontmatter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileHasFrontMatter(t *testing.T) {
	fm := func(filename string) bool {
		fm, err := FileHasFrontMatter(filename)
		require.NoError(t, err)
		return fm
	}
	require.True(t, fm("testdata/empty_fm.md"))
	require.True(t, fm("testdata/some_fm.md"))
	require.False(t, fm("testdata/no_fm.md"))
}

func TestFrontMatter_SortedStringArray(t *testing.T) {
	sorted := func(v interface{}) []string {
		fm := FrontMatter{"categories": v}
		return fm.SortedStringArray("categories")
	}
	require.Equal(t, []string{"a", "b"}, sorted("b a"))
	require.Equal(t, []string{"a", "b"}, sorted([]interface{}{"b", "a"}))
	require.Equal(t, []string{"a", "b"}, sorted([]string{"b", "a"}))
	require.Len(t, sorted(3), 0)
}
