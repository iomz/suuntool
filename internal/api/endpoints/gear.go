package endpoints

import (
	"context"
	"fmt"
	"strings"

	"github.com/tajchert/suuntool/internal/api"
)

// Gear represents a single piece of gear paired to the user's account.
type Gear struct {
	Serial       string  `json:"serial"`
	Nickname     string  `json:"nickname,omitempty"`
	Type         string  `json:"type,omitempty"`
	Distance     float64 `json:"distance,omitempty"`    // meters
	Duration     float64 `json:"duration,omitempty"`    // seconds
	Manufacturer string  `json:"manufacturer,omitempty"`
}

// GearList wraps all gear items.
type GearList struct {
	Items []Gear `json:"items"`
}

// Pretty returns one line per gear item: nickname (type) <serial> — <km> / <h:mm:ss>.
// If there are no items, returns "(no gear)".
func (g GearList) Pretty() string {
	if len(g.Items) == 0 {
		return "(no gear)"
	}
	var sb strings.Builder
	for _, item := range g.Items {
		name := item.Nickname
		if name == "" {
			name = item.Serial
		}
		typ := item.Type
		if typ != "" {
			typ = " (" + typ + ")"
		}
		fmt.Fprintf(&sb, "%s%s  %s — %s / %s\n",
			name, typ, item.Serial,
			formatKm(item.Distance),
			formatDuration(item.Duration),
		)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ListGear fetches all gear paired to the authenticated account.
func ListGear(ctx context.Context, c *api.Client) (*GearList, error) {
	body, err := c.Do(ctx, "GET", "gear", nil, nil)
	if err != nil {
		return nil, err
	}
	items, err := api.DecodeAsko[[]Gear](body)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []Gear{}
	}
	return &GearList{Items: items}, nil
}
