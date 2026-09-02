package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const personalAccessToken = "X-Personal-Access-Token"

type Client struct {
	BaseUrl    string
	Pat        string
	HttpClient *http.Client
}

func NewClient(baseUrl, pat string) *Client {
	client := &http.Client{Timeout: 10 * time.Minute}
	return &Client{
		BaseUrl:    strings.TrimRight(baseUrl, "/"),
		Pat:        pat,
		HttpClient: client,
	}
}

func (c *Client) req(method, path string, body io.Reader, headers map[string]string) (*http.Request, error) {
	r, err := http.NewRequest(method, c.BaseUrl+path, body)
	if err != nil {
		return nil, err
	}
	r.Header.Set(personalAccessToken, c.Pat)
	for k, v := range headers {
		r.Header.Set(k, v)
	}

	return r, nil
}

func (c *Client) do(method, path string, body io.Reader, headers map[string]string) ([]byte, error) {
	req, err := c.req(method, path, body, headers)
	if err != nil {
		return nil, err
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s: %s: %s", method, path, resp.Status, b)
	}

	return b, nil
}

func (c *Client) GetVersionSource(packageId, version string) ([]byte, error) {
	headers := map[string]string{"Accept": "application/octet-stream"}
	path := fmt.Sprintf("/api/v2/packages/%s/versions/%s/sources",
		url.PathEscape(packageId), url.PathEscape(version))
	return c.do(http.MethodGet, path, nil, headers)
}

func (c *Client) UpdateVersionSource(packageId, version string, zip []byte) error {
	headers := map[string]string{"Content-Type": "application/zip"}
	path := fmt.Sprintf("/api/v2/admin/packages/%s/versions/%s/sources",
		url.PathEscape(packageId), url.PathEscape(version))
	if _, err := c.do(http.MethodPut, path, bytes.NewReader(zip), headers); err != nil {
		return err
	}
	return nil
}
