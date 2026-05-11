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

// Tabular is implemented by list-shaped values that can produce a header row
// plus rows of cells. Used to render --format tsv (and reused by Pretty() for
// aligned text tables in endpoint code).
type Tabular interface {
	Table() (headers []string, rows [][]string)
}

// resolveFormat maps the requested format to "json", "pretty", or "tsv".
// For empty/"auto" it picks pretty on TTY, JSON otherwise.
func resolveFormat(f string, isTTY bool) string {
	switch strings.ToLower(f) {
	case "json":
		return "json"
	case "pretty":
		return "pretty"
	case "tsv":
		return "tsv"
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
	case "tsv":
		if t, ok := v.(Tabular); ok {
			return encodeTSV(w, t)
		}
		// Non-tabular values (single records, scalars) fall back to JSON so
		// pipelines still get something machine-parseable.
		return encodeJSON(w, v)
	default: // "json"
		return encodeJSON(w, v)
	}
}

// encodeTSV writes headers + rows as tab-separated values with \n line
// endings. Embedded tabs/newlines/carriage-returns in cells are replaced with
// a single space so each record stays on one line.
func encodeTSV(w io.Writer, t Tabular) error {
	headers, rows := t.Table()
	if _, err := fmt.Fprintln(w, joinTSV(headers)); err != nil {
		return err
	}
	for _, r := range rows {
		if _, err := fmt.Fprintln(w, joinTSV(r)); err != nil {
			return err
		}
	}
	return nil
}

var tsvScrubber = strings.NewReplacer("\t", " ", "\n", " ", "\r", " ")

func joinTSV(cells []string) string {
	scrubbed := make([]string, len(cells))
	for i, c := range cells {
		scrubbed[i] = tsvScrubber.Replace(c)
	}
	return strings.Join(scrubbed, "\t")
}

func encodeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// WriteRaw streams r to the given file path (truncating). Intended for binary or
// extremely large payloads (e.g. workout SML/FIT exports) that should bypass the
// json/pretty formatter. This is the ONE sanctioned bypass of Render — every other
// command must go through Render/RenderToFile via emit().
func WriteRaw(path string, r io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// WriteRawStdout streams r to os.Stdout. Same caveat as WriteRaw.
func WriteRawStdout(r io.Reader) error {
	_, err := io.Copy(os.Stdout, r)
	return err
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
		case ".tsv":
			resolved = "tsv"
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
