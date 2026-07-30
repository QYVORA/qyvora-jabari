package client

import (
	"crypto/tls"
	"net/http"
	"time"
)

type Option func(*Client)

type Client struct {
	httpClient *http.Client
	baseURL    string
	timeout    time.Duration
	headers    http.Header
	insecure   bool
}

func New(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{},
		timeout:    30 * time.Second,
		headers:    make(http.Header),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.httpClient.Timeout = c.timeout
	c.httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: c.insecure,
		},
	}

	return c
}

func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

func WithInsecure(v bool) Option {
	return func(c *Client) {
		c.insecure = v
	}
}

func WithHeader(key, value string) Option {
	return func(c *Client) {
		c.headers.Set(key, value)
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	for key, values := range c.headers {
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}
	return c.httpClient.Do(req)
}

func (c *Client) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}
