package output_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tajchert/suuntool/internal/output"
)

type sample struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

func (s sample) Pretty() string {
	return "name=" + s.Name
}

func TestRender_JSON_PrettyPrintsToWriter(t *testing.T) {
	var buf bytes.Buffer
	err := output.Render(&buf, sample{"x", 1}, output.Opts{Format: "json"})
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"name": "x"`)
}

func TestRender_Pretty_UsesPrettyMethod(t *testing.T) {
	var buf bytes.Buffer
	err := output.Render(&buf, sample{"x", 1}, output.Opts{Format: "pretty"})
	require.NoError(t, err)
	assert.Equal(t, "name=x\n", buf.String())
}

func TestRender_FormatFromExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	err := output.RenderToFile(path, sample{"x", 1}, output.Opts{Format: "auto"})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), `"name": "x"`))
}
