package endpoints

import "fmt"

// formatKm formats meters as a km string with two decimal places.
func formatKm(meters float64) string {
	return fmt.Sprintf("%.2fkm", meters/1000.0)
}

// formatDuration formats seconds as h:mm:ss.
func formatDuration(secs float64) string {
	total := int(secs)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}
