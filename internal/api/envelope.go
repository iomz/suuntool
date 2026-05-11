package api

import (
	"encoding/json"
	"fmt"
)

// AskoResponse is the standard envelope returned by /apiserver/* endpoints.
// `error == nil` means success.
type AskoResponse[T any] struct {
	Error    *AskoError      `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
	Payload  T               `json:"payload"`
}

type AskoError struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

// DecodeAsko parses body into an AskoResponse[T] and surfaces server errors as *Error.
func DecodeAsko[T any](body []byte) (T, error) {
	var resp AskoResponse[T]
	var zero T
	if err := json.Unmarshal(body, &resp); err != nil {
		return zero, &Error{Code: "BAD_ENVELOPE", Message: err.Error(), Exit: 5}
	}
	if resp.Error != nil {
		return zero, &Error{
			Code:    "SERVER",
			Message: fmt.Sprintf("asko error %d: %s", resp.Error.Code, resp.Error.Description),
			Exit:    5,
		}
	}
	return resp.Payload, nil
}
