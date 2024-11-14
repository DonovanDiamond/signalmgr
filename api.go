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

type errorResposne struct {
	Error string `json:"error"`
}

type params map[string]string

func encodeParams(params params) string {
	urlParams := url.Values{}
	for key, val := range params {
		urlParams.Add(key, val)
	}
	return urlParams.Encode()
}

// Sends a GET request to API_URL + path.
//
// Returns the response as raw bytes.
func getRaw(path string) (raw []byte, err error) {
	return completeRequest(fiber.Get(strings.TrimSuffix(API_URL, "/") + path))
}

// Sends a GET request to API_URL + path.
//
// JSON parses the response into resp of provided type.
func get[T any](path string) (resp T, err error) {
	raw, err := completeRequest(fiber.Get(strings.TrimSuffix(API_URL, "/") + path))
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &resp)
	return
}

// Sends a POST request to API_URL + path, parsing data into JSON as the body.
//
// JSON parses the response into resp of provided type.
func post[T any](path string, data any) (resp T, err error) {
	body, err := json.Marshal(data)
	if err != nil {
		return
	}
	req := fiber.Post(strings.TrimSuffix(API_URL, "/") + path)
	req.Body(body)
	req.ContentType("application/json")
	raw, err := completeRequest(req)
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &resp)
	return
}

// Sends a PUT request to API_URL + path, parsing data into JSON as the body.
//
// JSON parses the response into resp of provided type.
func put[T any](path string, data any) (resp T, err error) {
	body, err := json.Marshal(data)
	if err != nil {
		return
	}
	req := fiber.Put(strings.TrimSuffix(API_URL, "/") + path)
	req.Body(body)
	req.ContentType("application/json")
	raw, err := completeRequest(req)
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &resp)
	return
}

// Sends a DELETE request to API_URL + path, parsing data into JSON as the body.
//
// JSON parses the response into resp of provided type.
func delete[T any](path string, data any) (resp T, err error) {
	body, err := json.Marshal(data)
	if err != nil {
		return
	}
	req := fiber.Delete(strings.TrimSuffix(API_URL, "/") + path)
	req.Body(body)
	req.ContentType("application/json")
	raw, err := completeRequest(req)
	if err != nil {
		return
	}
	err = json.Unmarshal(raw, &resp)
	return
}

func completeRequest(req *fiber.Agent) (resp []byte, err error) {
	status, resp, errs := req.Bytes()
	if len(errs) > 0 {
		err = fmt.Errorf("%d errors on post request: %+v", len(errs), errs)
		return
	}
	var errResp errorResposne
	if json.Unmarshal(resp, &errResp); errResp.Error != "" {
		errResp.Error = strings.ReplaceAll(errResp.Error, "\n", "")
		err = errors.New(errResp.Error)
		return
	}
	if status < 200 || status > 299 {
		err = fmt.Errorf("status %d and response received: %s", status, resp)
		return
	}
	return
}
