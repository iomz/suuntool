package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tajchert/suuntool/internal/api"
)

// PartnerConnections is returned as a fairly loose shape — the server includes
// per-partner OAuth URLs that change format over time. Keep as RawMessage so
// JSON output is faithful; pretty rendering surfaces top-level partner names.
type PartnerConnections struct {
	Partners json.RawMessage `json:"-"`
}

// Partner is a single normalized row used for pretty rendering only.
type Partner struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
}

// Pretty returns one line per partner: "<name>   connected=<bool>".
// Falls back to indented JSON if the shape has changed.
func (p PartnerConnections) Pretty() string {
	var rows []Partner
	if err := json.Unmarshal(p.Partners, &rows); err == nil {
		var sb strings.Builder
		for _, r := range rows {
			fmt.Fprintf(&sb, "%s\tconnected=%v\n", r.Name, r.Connected)
		}
		return strings.TrimRight(sb.String(), "\n")
	}
	// Fallback: pretty-print the raw JSON.
	var v interface{}
	if err := json.Unmarshal(p.Partners, &v); err != nil {
		return string(p.Partners)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(p.Partners)
	}
	return string(out)
}

// MarshalJSON keeps the raw shape (faithful) for --format=json.
func (p PartnerConnections) MarshalJSON() ([]byte, error) {
	if len(p.Partners) == 0 {
		return []byte("null"), nil
	}
	return p.Partners, nil
}

// Partners fetches the partner connections for the authenticated user.
func Partners(ctx context.Context, c *api.Client) (*PartnerConnections, error) {
	body, err := c.Do(ctx, "GET", "partnerconnection", nil, nil)
	if err != nil {
		return nil, err
	}
	raw, err := api.DecodeAsko[json.RawMessage](body)
	if err != nil {
		return nil, err
	}
	return &PartnerConnections{Partners: raw}, nil
}
