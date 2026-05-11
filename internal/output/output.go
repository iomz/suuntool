package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Opts controls how output is rendered.
type Opts struct {
	Format string // "json", "pretty", "auto", or ""
	IsTTY  bool
}

// Prettier is implemented by values that know how to produce human-readable output.
type Prettier interface {
	Pretty() string
}

// resolveFormat maps the requested format to either "json" or "pretty".
// For empty/"auto" it picks based on IsTTY.
func resolveFormat(f string, isTTY bool) string {
	switch strings.ToLower(f) {
	case "json":
		return "json"
	case "pretty":
		return "pretty"
	default:
		// "auto" or ""
		if isTTY {
			return "pretty"
		}
		return "json"
	}
}

// Render writes v to w in the format specified by opts.
func Render(w io.Writer, v any, opts Opts) error {
	switch resolveFormat(opts.Format, opts.IsTTY) {
	case "pretty":
		if p, ok := v.(Prettier); ok {
			_, err := fmt.Fprintln(w, p.Pretty())
			return err
		}
		// Fall back to indented JSON when no Pretty() method.
		return encodeJSON(w, v)
	default: // "json"
		return encodeJSON(w, v)
	}
}

func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// RenderToFile writes v to the file at path.
// When opts.Format is empty or "auto", the format is inferred from the file
// extension (.json → "json"; anything else → "json" as default).
// Parent directories are created with mode 0755; the file is written with 0644.
func RenderToFile(path string, v any, opts Opts) error {
	resolved := opts.Format
	if resolved == "" || strings.ToLower(resolved) == "auto" {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".json":
			resolved = "json"
		default:
			resolved = "json"
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	return Render(f, v, Opts{Format: resolved, IsTTY: false})
}
