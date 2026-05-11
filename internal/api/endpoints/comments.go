package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tajchert/suuntool/internal/api"
)

// Comment is one entry returned by /v1/workouts/comments/{key}.
type Comment struct {
	Key       string `json:"key"`
	Comment   string `json:"comment"`
	Username  string `json:"username"`
	Timestamp int64  `json:"timestamp"` // unix ms
}

// Pretty returns a single formatted line for the comment.
func (c Comment) Pretty() string {
	ts := time.Unix(0, c.Timestamp*int64(time.Millisecond)).UTC().Format(time.RFC3339)
	username := c.Username
	if username == "" {
		username = "(unknown)"
	}
	return fmt.Sprintf("%s  %s: %s  [%s]", ts, username, c.Comment, c.Key)
}

// CommentList wraps the comments slice for list output.
type CommentList struct {
	Items []Comment `json:"items"`
}

// Pretty returns one line per comment plus a footer with count.
func (l CommentList) Pretty() string {
	var sb strings.Builder
	for _, c := range l.Items {
		sb.WriteString(c.Pretty())
		sb.WriteByte('\n')
	}
	fmt.Fprintf(&sb, "(%d comments)", len(l.Items))
	return sb.String()
}

// ListComments fetches /v1/workouts/comments/{workoutKey}.
func ListComments(ctx context.Context, c *api.Client, workoutKey string) (*CommentList, error) {
	body, err := c.Do(ctx, "GET", "workouts/comments/"+workoutKey, nil, nil)
	if err != nil {
		return nil, err
	}
	items, err := api.DecodeAsko[[]Comment](body)
	if err != nil {
		return nil, err
	}
	return &CommentList{Items: items}, nil
}

// PostComment writes a comment to /v1/workouts/comment/{workoutKey}.
// The body is sent as text/plain. The caller must supply the x-totp header
// (generated via totpHeaders in cmd/root.go) in the headers map.
// Returns the raw response payload so the caller can emit it without
// redefining the workout type here.
func PostComment(ctx context.Context, c *api.Client, workoutKey, text string, headers map[string]string) (json.RawMessage, error) {
	bodyReader, contentTypeHdr := api.TextBody(text)
	merged := make(map[string]string, len(headers)+len(contentTypeHdr))
	for k, v := range headers {
		merged[k] = v
	}
	for k, v := range contentTypeHdr {
		merged[k] = v
	}
	respBody, err := c.Do(ctx, "POST", "workouts/comment/"+workoutKey, bodyReader, merged)
	if err != nil {
		return nil, err
	}
	return api.DecodeAsko[json.RawMessage](respBody)
}

// DeleteComment removes a comment by its comment key (not workout key).
func DeleteComment(ctx context.Context, c *api.Client, commentKey string) error {
	_, err := c.Do(ctx, "DELETE", "workouts/comment/"+commentKey, nil, nil)
	return err
}
