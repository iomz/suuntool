package endpoints

import (
	"context"
	"fmt"
	"strings"

	"github.com/tajchert/suuntool/internal/api"
)

// BBox is the bounding box for a map region.
type BBox struct {
	LowerLat float64 `json:"lowerLat"`
	LowerLng float64 `json:"lowerLng"`
	UpperLat float64 `json:"upperLat"`
	UpperLng float64 `json:"upperLng"`
}

// MapRegion represents a single offline map region.
type MapRegion struct {
	RegionID     string `json:"regionId"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
	Size         int64  `json:"size,omitempty"`
	TransferSize int64  `json:"transferSize,omitempty"`
	BoundaryURL  string `json:"boundaryUrl,omitempty"`
	MaskURL      string `json:"maskUrl,omitempty"`
	BBox         *BBox  `json:"bbox,omitempty"`
}

// MapLibrary wraps all map regions.
type MapLibrary struct {
	Items []MapRegion `json:"items"`
}

// Pretty returns one line per region: <regionId>  <name>  <status>  <size>MB.
func (m MapLibrary) Pretty() string {
	if len(m.Items) == 0 {
		return "(no maps)"
	}
	var sb strings.Builder
	for _, r := range m.Items {
		sizeMB := float64(r.Size) / (1024 * 1024)
		fmt.Fprintf(&sb, "%s\t%s\t%s\t%.1fMB\n", r.RegionID, r.Name, r.Status, sizeMB)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ListMaps fetches the offline-map library for a specific Suunto device serial.
// The serial number can be found in /sml responses as Source: "suunto-<sn>".
func ListMaps(ctx context.Context, c *api.Client, deviceSerial string) (*MapLibrary, error) {
	if deviceSerial == "" {
		return nil, &api.Error{
			Code:    "USAGE",
			Message: "deviceSerial is required",
			Hint:    "Pass --device-serial <sn>",
			Exit:    2,
		}
	}
	body, err := c.Do(ctx, "GET", "maps/library?deviceSerialNumber="+deviceSerial, nil, nil)
	if err != nil {
		return nil, err
	}
	items, err := api.DecodeAsko[[]MapRegion](body)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []MapRegion{}
	}
	return &MapLibrary{Items: items}, nil
}
