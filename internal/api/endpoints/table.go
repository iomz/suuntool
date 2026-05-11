package endpoints

import (
	"strings"
	"unicode/utf8"
)

// renderTable returns a fixed-width text table. Aligns each column to its
// widest cell (measured in runes, not bytes); two spaces between columns; an
// ─ underline separates header from rows. No trailing newline.
func renderTable(headers []string, rows [][]string) string {
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
	writeRow := func(cells []string) {
		for i, c := range cells {
			if i > 0 {
				sb.WriteString("  ")
			}
			sb.WriteString(c)
			if pad := widths[i] - utf8.RuneCountInString(c); pad > 0 {
				sb.WriteString(strings.Repeat(" ", pad))
			}
		}
		sb.WriteByte('\n')
	}
	writeRow(headers)
	for i, w := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("─", w))
	}
	sb.WriteByte('\n')
	for _, r := range rows {
		writeRow(r)
	}
	return strings.TrimRight(sb.String(), "\n")
}
