package lyricfetch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const maxResponseBytes = 2_000_000

func (c *Client) request(ctx context.Context, method, endpoint string, params url.Values, body any, headers map[string]string) ([]byte, int, error) {
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}
	var input io.Reader
	if form, ok := body.(url.Values); ok {
		input = strings.NewReader(form.Encode())
	} else if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, 0, failure(InvalidResponse, "cannot encode request")
		}
		input = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, input)
	if err != nil {
		return nil, 0, failure(Unavailable, "cannot construct request")
	}
	req.Header.Set("User-Agent", "lyricfetch/0.1 (+https://github.com/s04/lyricfetch)")
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		if _, ok := body.(url.Values); ok {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return nil, 0, transportError(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, response.StatusCode, failure(Unavailable, fmt.Sprintf("HTTP %d", response.StatusCode))
	}
	if response.ContentLength > maxResponseBytes {
		return nil, response.StatusCode, failure(InvalidResponse, "response too large")
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return nil, response.StatusCode, transportError(err)
	}
	if len(data) > maxResponseBytes {
		return nil, response.StatusCode, failure(InvalidResponse, "response too large")
	}
	return data, response.StatusCode, nil
}

func transportError(err error) error {
	if errors.Is(err, context.Canceled) {
		return failure(Canceled, "request canceled")
	}
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		return failure(TimedOut, "request timed out")
	}
	return failure(Unavailable, "request failed")
}

func getJSON[T any](c *Client, ctx context.Context, endpoint string, params url.Values, missingOK bool) (T, bool, error) {
	var value T
	data, status, err := c.request(ctx, http.MethodGet, endpoint, params, nil, nil)
	if missingOK && status == http.StatusNotFound {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	if err = decodeJSON(data, &value); err != nil {
		return value, false, err
	}
	return value, true, nil
}

func decodeJSON(data []byte, value any) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return failure(InvalidResponse, "unexpected null response")
	}
	if err := json.Unmarshal(data, value); err != nil {
		return failure(InvalidResponse, "invalid JSON response")
	}
	return nil
}

func apiStatus(value *int, want int) error {
	if value == nil {
		return failure(InvalidResponse, "missing API status")
	}
	if *value != want {
		return failure(Unavailable, fmt.Sprintf("API status %d", *value))
	}
	return nil
}
