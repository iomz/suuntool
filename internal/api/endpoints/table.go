package endpoints

import (
	"strings"
	"unicode/utf8"
)

// renderTable returns a fixed-width text table. Aligns each column to its
// widest cell (measured in runes, not bytes); two spaces between columns; an
// ─ underline separates header from rows. No trailing newline.
func renderTable(headers []string, rows [][]string) string {
	return renderTableStyled(headers, rows, nil)
}

// cellStyler returns a styled version of a plain cell. row == -1 marks the
// header. Widths are measured against the plain text before styling, so styled
// strings can include zero-width ANSI escapes without breaking alignment.
type cellStyler func(col, row int, plain string) string

// renderTableStyled is renderTable plus an optional per-cell styler. When
// style is nil, output is identical to renderTable.
func renderTableStyled(headers []string, rows [][]string, style cellStyler) string {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i >= len(widths) {
				break
			}
			if n := utf8.RuneCountInString(c); n > widths[i] {
				widths[i] = n
			}
		}
	}
	var sb strings.Builder
	writeCell := func(col, row int, plain string) {
		if col > 0 {
			sb.WriteString("  ")
		}
		out := plain
		if style != nil {
			out = style(col, row, plain)
		}
		sb.WriteString(out)
		if pad := widths[col] - utf8.RuneCountInString(plain); pad > 0 {
			sb.WriteString(strings.Repeat(" ", pad))
		}
	}
	for i, h := range headers {
		writeCell(i, -1, h)
	}
	sb.WriteByte('\n')
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("─", w))
	}
	sb.WriteByte('\n')
	for ri, r := range rows {
		for ci, c := range r {
			writeCell(ci, ri, c)
		}
		sb.WriteByte('\n')
	}
	return strings.TrimRight(sb.String(), "\n")
}
