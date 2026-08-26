package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type selfCheckClient struct {
	baseURL string
	client  *http.Client
	serial  int
}

func newSelfCheckClient(address string) *selfCheckClient {
	return &selfCheckClient{
		baseURL: "http://" + address,
		client:  &http.Client{Timeout: 4 * time.Second},
	}
}

func (c *selfCheckClient) get(ctx context.Context, path string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	return c.execute(request, http.StatusOK, target)
}

func (c *selfCheckClient) post(ctx context.Context, path, role string, body, target any, expectedStatus int) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.serial++
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Actor-Role", role)
	request.Header.Set("Idempotency-Key", fmt.Sprintf("selfcheck-%03d", c.serial))
	return c.execute(request, expectedStatus, target)
}

func (c *selfCheckClient) execute(request *http.Request, expectedStatus int, target any) error {
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != expectedStatus {
		return fmt.Errorf("%s %s 返回 %d，响应: %s", request.Method, request.URL.Path, response.StatusCode, string(payload))
	}
	if target != nil {
		if err := json.Unmarshal(payload, target); err != nil {
			return fmt.Errorf("解码 %s 响应: %w", request.URL.Path, err)
		}
	}
	return nil
}
