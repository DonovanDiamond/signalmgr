package signalmgr

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var API_URL = "http://127.0.0.1:8080"

// APIError is returned for any non-2xx response. Account and ChallengeTokens
// are only set by endpoints that report them, e.g. a rate-limited /v2/send.
type APIError struct {
	Method          string   `json:"-"`
	Path            string   `json:"-"`
	Status          int      `json:"-"`
	Message         string   `json:"error"`
	Account         string   `json:"account"`
	ChallengeTokens []string `json:"challenge_tokens"`

	// Set when Message is the raw body because it had no JSON error field.
	rawBody bool
}

func (e *APIError) Error() string {
	if e.rawBody {
		return fmt.Sprintf("%s %s: status %d and response received: %s", e.Method, e.Path, e.Status, e.Message)
	}
	return fmt.Sprintf("%s %s: status %d: %s", e.Method, e.Path, e.Status, strings.ReplaceAll(e.Message, "\n", ""))
}

// Appends q as a query string to path, or returns path unchanged when q is empty.
func withQuery(path string, q url.Values) string {
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

var methods = map[string]func(string) *fiber.Agent{
	fiber.MethodGet:    fiber.Get,
	fiber.MethodPost:   fiber.Post,
	fiber.MethodPut:    fiber.Put,
	fiber.MethodDelete: fiber.Delete,
}

// Sends a request to API_URL + path. A non-nil data is sent as the JSON body;
// nil sends no body.
//
// Returns the response as raw bytes.
func request(method, path string, data any) (raw []byte, err error) {
	req := methods[method](strings.TrimSuffix(API_URL, "/") + path)
	// The client overwrites the per-request flag, so set it here to keep
	// escaped segments such as %2F in group IDs from being decoded.
	if req.HostClient != nil { // nil when API_URL failed to parse; Bytes reports that error
		req.HostClient.DisablePathNormalizing = true
	}
	if data != nil {
		body, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		req.Body(body)
		req.ContentType("application/json")
	}
	status, raw, errs := req.Bytes()
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s %s: %w", method, path, errors.Join(errs...))
	}
	if status < 200 || status > 299 {
		// Only error statuses carry an error body.
		apiErr := &APIError{Method: method, Path: path, Status: status}
		json.Unmarshal(raw, apiErr)
		if apiErr.Message == "" {
			apiErr.Message, apiErr.rawBody = string(raw), true
		}
		return nil, apiErr
	}
	return raw, nil
}

func requestJSON[T any](method, path string, data any) (resp T, err error) {
	raw, err := request(method, path, data)
	if err != nil || len(raw) == 0 {
		return
	}
	err = json.Unmarshal(raw, &resp)
	return
}

// Sends a GET request to API_URL + path.
//
// Returns the response as raw bytes.
func getRaw(path string) (raw []byte, err error) {
	return request(fiber.MethodGet, path, nil)
}

// Sends a GET request to API_URL + path.
//
// JSON parses the response into resp of provided type.
func get[T any](path string) (resp T, err error) {
	return requestJSON[T](fiber.MethodGet, path, nil)
}

// Sends a POST request to API_URL + path, parsing data into JSON as the body.
//
// JSON parses the response into resp of provided type.
func post[T any](path string, data any) (resp T, err error) {
	return requestJSON[T](fiber.MethodPost, path, data)
}

// Sends a PUT request to API_URL + path, parsing data into JSON as the body.
//
// JSON parses the response into resp of provided type.
func put[T any](path string, data any) (resp T, err error) {
	return requestJSON[T](fiber.MethodPut, path, data)
}

// Sends a DELETE request to API_URL + path, parsing data into JSON as the body.
//
// JSON parses the response into resp of provided type.
func delete[T any](path string, data any) (resp T, err error) {
	return requestJSON[T](fiber.MethodDelete, path, data)
}
