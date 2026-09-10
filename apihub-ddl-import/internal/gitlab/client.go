// Package gitlab is a minimal GitLab REST v4 client: list a repository tree and
// download raw files. Works for gitlab.com and self-hosted instances alike.
package gitlab

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const tokenHeader = "PRIVATE-TOKEN"

type Client struct {
	APIBase    string // https://host/api/v4
	Project    string // URL-encoded full path, e.g. group%2Fsub%2Frepo
	Token      string
	HTTPClient *http.Client
}

// New builds a client from a repository URL like https://gitlab.com/group/sub/repo(.git).
// Note: the project path must be encoded with url.QueryEscape — url.PathEscape
// keeps "/" unescaped, which GitLab rejects.
func New(repoURL, token string, insecureSkipTLSVerify bool) (*Client, error) {
	u, err := url.Parse(strings.TrimRight(repoURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse repo url %q: %w", repoURL, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("repo url %q must be absolute (https://host/group/repo)", repoURL)
	}
	project := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
	if project == "" {
		return nil, fmt.Errorf("repo url %q has no project path", repoURL)
	}
	hc := &http.Client{Timeout: 120 * time.Second}
	if insecureSkipTLSVerify {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in flag for private PKI
		hc.Transport = tr
	}
	return &Client{
		APIBase:    u.Scheme + "://" + u.Host + "/api/v4",
		Project:    url.QueryEscape(project),
		Token:      token,
		HTTPClient: hc,
	}, nil
}

func (c *Client) get(path string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, c.APIBase+path, nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set(tokenHeader, c.Token)
	}
	return c.HTTPClient.Do(req)
}

// TreeEntry is one item of the repository tree listing.
type TreeEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // blob | tree
	Path string `json:"path"`
}

// ListTree lists the repository tree under path at ref, recursively, walking
// GitLab pagination.
func (c *Client) ListTree(treePath, ref string) ([]TreeEntry, error) {
	var all []TreeEntry
	const perPage = 100
	page := 1
	for {
		q := url.Values{}
		q.Set("ref", ref)
		q.Set("recursive", "true")
		q.Set("per_page", strconv.Itoa(perPage))
		q.Set("page", strconv.Itoa(page))
		if treePath != "" {
			q.Set("path", treePath)
		}
		resp, err := c.get("/projects/" + c.Project + "/repository/tree?" + q.Encode())
		if err != nil {
			return nil, fmt.Errorf("gitlab list tree: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("gitlab list tree %q ref %q: status %d: %s", treePath, ref, resp.StatusCode, excerpt(body))
		}
		var chunk []TreeEntry
		if err := json.Unmarshal(body, &chunk); err != nil {
			return nil, fmt.Errorf("gitlab list tree decode: %w", err)
		}
		all = append(all, chunk...)
		next := resp.Header.Get("X-Next-Page")
		if next == "" || next == "0" {
			if len(chunk) < perPage {
				break
			}
			page++
			continue
		}
		p, err := strconv.Atoi(next)
		if err != nil {
			break
		}
		page = p
	}
	return all, nil
}

// GetRawFile downloads a file's raw content at ref.
func (c *Client) GetRawFile(filePath, ref string) ([]byte, error) {
	p := "/projects/" + c.Project + "/repository/files/" + url.QueryEscape(strings.Trim(filePath, "/")) +
		"/raw?ref=" + url.QueryEscape(ref)
	resp, err := c.get(p)
	if err != nil {
		return nil, fmt.Errorf("gitlab get file %s: %w", filePath, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gitlab get file %q ref %q: status %d: %s", filePath, ref, resp.StatusCode, excerpt(body))
	}
	return body, nil
}

func excerpt(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
