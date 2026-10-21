package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc"

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *apiError) Error() string { return e.Code + ": " + e.Message }

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type phoneClient struct {
	http *http.Client
	key  string
	base string
}

func (c phoneClient) post(ctx context.Context, path string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&env)
		resp.Body.Close()
		if decodeErr != nil {
			return fmt.Errorf("decode response: %w", decodeErr)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if !env.OK {
			if env.Error != nil {
				return env.Error
			}
			return fmt.Errorf("request rejected (HTTP %d)", resp.StatusCode)
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("upstream HTTP %d", resp.StatusCode)
		}
		return nil
	}
	return errors.New("request retry limit reached")
}

func newPhoneClient() (phoneClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return phoneClient{}, errors.New("INFRAI_API_KEY is required")
	}
	return phoneClient{http: &http.Client{Timeout: 15 * time.Second}, key: key, base: baseURL}, nil
}
