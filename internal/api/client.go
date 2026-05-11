package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tajchert/suuntool/internal/auth"
)

const DefaultBaseURL = "https://api.sports-tracker.com/apiserver/v1/"

// Client is a thin HTTP client that injects STTAuthorization and maps HTTP
// status codes to typed errors. Construct via NewClient.
type Client struct {
	BaseURL    string
	HTTP       *http.Client
	SessionKey string // empty for unauthenticated calls
}

func NewClient(baseURL, sessionKey string, timeout time.Duration) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &Client{
		BaseURL:    baseURL,
		SessionKey: sessionKey,
		HTTP:       &http.Client{Timeout: timeout},
	}
}

// Do executes the request, applies common headers, and maps HTTP errors to *Error.
// On 2xx it returns the raw body. Envelope decoding is the caller's responsibility
// (DecodeAsko for ASKO endpoints).
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
	u, err := c.resolve(path)
	if err != nil {
		return nil, &Error{Code: "USAGE", Message: err.Error(), Exit: 2}
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, &Error{Code: "USAGE", Message: err.Error(), Exit: 2}
	}
	req.Header.Set("User-Agent", auth.UserAgent)
	req.Header.Set("Accept-Language", "en")
	if c.SessionKey != "" {
		req.Header.Set("STTAuthorization", c.SessionKey)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			return nil, &Error{Code: "NETWORK", Message: "timeout: " + err.Error(), Exit: 3}
		}
		return nil, &Error{Code: "NETWORK", Message: err.Error(), Exit: 3}
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return b, nil
	case resp.StatusCode == 401:
		return nil, &Error{Code: "AUTH_EXPIRED", Message: "session rejected", Hint: "Run: suuntool login", HTTP: 401, Exit: 4}
	case resp.StatusCode == 403:
		return nil, &Error{Code: "FORBIDDEN", Message: snippet(b), HTTP: 403, Exit: 7}
	case resp.StatusCode == 404:
		return nil, &Error{Code: "NOT_FOUND", Message: snippet(b), HTTP: 404, Exit: 6}
	case resp.StatusCode >= 500:
		return nil, &Error{Code: "SERVER", Message: snippet(b), HTTP: resp.StatusCode, Exit: 5}
	default:
		return nil, &Error{Code: "SERVER", Message: fmt.Sprintf("unexpected HTTP %d: %s", resp.StatusCode, snippet(b)), HTTP: resp.StatusCode, Exit: 5}
	}
}

func (c *Client) resolve(path string) (string, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path, nil
	}
	u, err := url.Parse(c.BaseURL + strings.TrimPrefix(path, "/"))
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func snippet(b []byte) string {
	const max = 200
	if len(b) > max {
		return string(b[:max]) + "…"
	}
	return string(b)
}
