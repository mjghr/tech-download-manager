package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

type HTTPClient struct {
	client *http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{},
	}
}

func (c *HTTPClient) MakeRequest(req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *HTTPClient) NewRequest(method, url string, headers map[string]string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if body != nil {
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return req, nil
}

func (c *HTTPClient) NewRequestWithContext(ctx context.Context, method, url string, headers map[string]string, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if body != nil {
		req.Body = io.NopCloser(bytes.NewBuffer(body))
	}
	return req, nil
}

func (c *HTTPClient) SendRequest(method string, url string, headers map[string]string) (*http.Response, error) {
	req, err := c.NewRequest(method, url, headers, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *HTTPClient) SendRequestWithContext(ctx context.Context, method string, url string, headers map[string]string) (*http.Response, error) {
	req, err := c.NewRequestWithContext(ctx, method, url, headers, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.MakeRequest(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}