package apihub

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const APIKeyHeader = "api-key"

const versionsListPageLimit = 100 // must match backend getLimitQueryParam default max slice

// VerboseHTTP enables extra HTTP diagnostics when HTTPDebug is set (e.g. from --debug).
var VerboseHTTP bool

// HTTPDebug receives formatted lines (typically stderr); nil disables logging.
var HTTPDebug func(format string, args ...any)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// New returns an API client. If insecureSkipTLSVerify is true, TLS server certificates are not
// verified (susceptible to MITM); use only with private CAs or troubleshooting.
func New(baseURL, apiKey string, insecureSkipTLSVerify bool) *Client {
	hc := &http.Client{Timeout: 600 * time.Second}
	if insecureSkipTLSVerify {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in flag for dev / private PKI
		hc.Transport = tr
	}
	return &Client{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		APIKey:     apiKey,
		HTTPClient: hc,
	}
}
func (c *Client) req(method, path string, body io.Reader, headers map[string]string) (*http.Request, error) {
	u := c.BaseURL + path
	r, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, err
	}
	r.Header.Set(APIKeyHeader, c.APIKey)
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r, nil
}

func httpDebugLog(method, path string, status int, body []byte) {
	if !VerboseHTTP || HTTPDebug == nil {
		return
	}
	msg := summarizeBodyForDebug(body)
	HTTPDebug("HTTP %-6s %-80s ← %d  %s", method, truncatePathForLog(path), status, msg)
}

func summarizeBodyForDebug(body []byte) string {
	switch {
	case len(body) == 0:
		return "(no body)"
	case len(body) > 6144 || looksBinary(body):
		return fmt.Sprintf("(%d bytes, binary/non-text omit)", len(body))
	default:
		s := strings.TrimSpace(string(body))
		if len(s) > 4000 {
			return fmt.Sprintf("%d chars: %s… [truncated]", len(body), s[:4000])
		}
		return fmt.Sprintf("%d chars: %s", len(body), s)
	}
}

func looksBinary(b []byte) bool {
	n := len(b)
	if n > 512 {
		n = 512
	}
	for i := 0; i < n; i++ {
		if b[i] == 0 {
			return true
		}
		if b[i] < 32 && b[i] != '\n' && b[i] != '\r' && b[i] != '\t' {
			return true
		}
	}
	return false
}

func truncatePathForLog(path string) string {
	if len(path) <= 96 {
		return path
	}
	return path[:45] + "…" + path[len(path)-45:]
}

func (c *Client) doJSON(method, path string, body io.Reader, headers map[string]string, out any) (int, []byte, error) {
	r, err := c.req(method, path, body, headers)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		if headers == nil || headers["Content-Type"] == "" {
			r.Header.Set("Content-Type", "application/json")
		}
	}
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		if VerboseHTTP && HTTPDebug != nil {
			HTTPDebug("HTTP %s %s → error: %v", method, truncatePathForLog(path), err)
		}
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		httpDebugLog(method, path, resp.StatusCode, nil)
		return resp.StatusCode, nil, err
	}
	httpDebugLog(method, path, resp.StatusCode, b)
	if out != nil && len(b) > 0 && resp.StatusCode < 300 {
		if err := json.Unmarshal(b, out); err != nil {
			return resp.StatusCode, b, err
		}
	}
	return resp.StatusCode, b, nil
}

type PackageInfo struct {
	PackageId   string              `json:"packageId"`
	Alias       string              `json:"alias"`
	ParentId    string              `json:"parentId"`
	Kind        string              `json:"kind"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Parents     []ParentPackageInfo `json:"parents"`
}

type ParentPackageInfo struct {
	PackageId string `json:"packageId"`
	Kind      string `json:"kind"`
}

type PackagesList struct {
	Packages []PackagesInfo `json:"packages"`
}

type PackagesInfo struct {
	PackageId string `json:"packageId"`
	Alias     string `json:"alias"`
	ParentId  string `json:"parentId"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
}

type PublishedVersionsView struct {
	Versions []PublishedVersionListView `json:"versions"`
}

type PublishedVersionListView struct {
	Version string `json:"version"`
}

type PublishStatusResponse struct {
	PublishId string `json:"publishId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

type PublishAccepted struct {
	PublishId string `json:"publishId"`
}

func packagePath(packageID string) string {
	return "/api/v2/packages/" + url.PathEscape(packageID)
}

// GetPackage loads package metadata (showParents=true).
func (c *Client) GetPackage(packageID string) (*PackageInfo, int, error) {
	path := packagePath(packageID) + "?showParents=true"
	var p PackageInfo
	code, raw, err := c.doJSON(http.MethodGet, path, nil, nil, &p)
	if err != nil {
		return nil, code, fmt.Errorf("get package decode: %w body=%s", err, string(raw))
	}
	if code != http.StatusOK {
		return nil, code, fmt.Errorf("get package: status %d: %s", code, string(raw))
	}
	return &p, code, nil
}

// ListPackagesPage lists children or all descendants under parentID.
func (c *Client) ListPackagesPage(parentID string, showAllDescendants bool, kind string, page, limit int) (*PackagesList, int, error) {
	q := url.Values{}
	q.Set("parentId", parentID)
	if showAllDescendants {
		q.Set("showAllDescendants", "true")
	}
	if kind != "" {
		q.Set("kind", kind)
	}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	path := "/api/v2/packages?" + q.Encode()
	var out PackagesList
	code, raw, err := c.doJSON(http.MethodGet, path, nil, nil, &out)
	if err != nil {
		return nil, code, fmt.Errorf("list packages decode: %w body=%s", err, string(raw))
	}
	if code != http.StatusOK {
		return nil, code, fmt.Errorf("list packages: status %d: %s", code, string(raw))
	}
	return &out, code, nil
}

// ListVersions returns all published versions for a package by walking API pagination (default page size 100).
func (c *Client) ListVersions(packageID string) (*PublishedVersionsView, int, error) {
	var merged []PublishedVersionListView
	page := 0
	lastCode := http.StatusOK
	for {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(versionsListPageLimit))
		path := "/api/v3/packages/" + url.PathEscape(packageID) + "/versions?" + q.Encode()
		var chunk PublishedVersionsView
		code, raw, err := c.doJSON(http.MethodGet, path, nil, nil, &chunk)
		lastCode = code
		if err != nil {
			return nil, code, fmt.Errorf("list versions pkg=%s page=%d decode: %w body=%s", packageID, page, err, string(raw))
		}
		if code != http.StatusOK {
			return nil, code, fmt.Errorf("list versions pkg=%s page=%d: status %d: %s", packageID, page, code, string(raw))
		}
		merged = append(merged, chunk.Versions...)
		if len(chunk.Versions) < versionsListPageLimit {
			break
		}
		page++
		if page > 1000 {
			return nil, lastCode, fmt.Errorf("list versions pkg=%s: pagination safety stop after %d pages", packageID, page)
		}
	}
	return &PublishedVersionsView{Versions: merged}, lastCode, nil
}

// GetVersionSources downloads the original published sources zip.
func (c *Client) GetVersionSources(packageID, version string) ([]byte, int, error) {
	path := fmt.Sprintf("/api/v2/packages/%s/versions/%s/sources", url.PathEscape(packageID), url.PathEscape(version))
	r, err := c.req(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode != http.StatusOK {
		httpDebugLog(http.MethodGet, path+" (sources zip)", resp.StatusCode, b)
		return b, resp.StatusCode, fmt.Errorf("get sources: status %d: %s", resp.StatusCode, string(b))
	}
	if VerboseHTTP && HTTPDebug != nil {
		HTTPDebug("HTTP %-6s %-80s ← %d  (zip payload %d bytes)", http.MethodGet, truncatePathForLog(path), resp.StatusCode, len(b))
	}
	return b, resp.StatusCode, nil
}

// GetVersionBuildConfigJSON returns raw JSON for /config (publish config snapshot).
func (c *Client) GetVersionBuildConfigJSON(packageID, version string) (json.RawMessage, int, error) {
	path := fmt.Sprintf("/api/v2/packages/%s/versions/%s/config", url.PathEscape(packageID), url.PathEscape(version))
	code, raw, err := c.doJSON(http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return nil, code, err
	}
	if code != http.StatusOK {
		return nil, code, fmt.Errorf("get config: status %d: %s", code, string(raw))
	}
	return json.RawMessage(raw), code, nil
}

type CreatePackageBody struct {
	Alias       string `json:"alias"`
	ParentId    string `json:"parentId"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (c *Client) CreatePackage(body CreatePackageBody) (*PackageInfo, int, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, 0, err
	}
	path := "/api/v2/packages"
	var p PackageInfo
	code, raw, err := c.doJSON(http.MethodPost, path, bytes.NewReader(buf), map[string]string{"Content-Type": "application/json"}, &p)
	if code != http.StatusCreated {
		return nil, code, fmt.Errorf("create package: status %d: %s", code, string(raw))
	}
	if err != nil {
		return nil, code, fmt.Errorf("create package decode: %w body=%s", err, string(raw))
	}
	return &p, code, nil
}

// PublishVersion posts multipart publish. If sync=true, HTTP 204 — no polling.
func (c *Client) PublishVersion(packageID string, sourcesZip, configJSON []byte, resolveRefs, resolveConflicts bool) (publishID string, sync bool, code int, err error) {
	path := packagePath(packageID) + "/publish"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormField("config")
	if err != nil {
		return "", false, 0, err
	}
	if _, err := fw.Write(configJSON); err != nil {
		return "", false, 0, err
	}
	part, err := w.CreateFormFile("sources", "sources.zip")
	if err != nil {
		return "", false, 0, err
	}
	if _, err := part.Write(sourcesZip); err != nil {
		return "", false, 0, err
	}
	_ = w.WriteField("resolveRefs", strconv.FormatBool(resolveRefs))
	_ = w.WriteField("resolveConflicts", strconv.FormatBool(resolveConflicts))
	if err := w.Close(); err != nil {
		return "", false, 0, err
	}
	r, err := c.req(http.MethodPost, path, &b, map[string]string{"Content-Type": w.FormDataContentType()})
	if err != nil {
		return "", false, 0, err
	}
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return "", false, 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	code = resp.StatusCode
	httpDebugLog(http.MethodPost, path+" (multipart publish)", code, raw)
	switch code {
	case http.StatusNoContent:
		return "", true, code, nil
	case http.StatusAccepted:
		var acc PublishAccepted
		if err := json.Unmarshal(raw, &acc); err != nil {
			return "", false, code, fmt.Errorf("decode publish response: %w body=%s", err, string(raw))
		}
		return acc.PublishId, false, code, nil
	default:
		return "", false, code, fmt.Errorf("publish: status %d: %s", code, string(raw))
	}
}

func (c *Client) GetPublishStatus(packageID, publishID string) (*PublishStatusResponse, int, error) {
	path := fmt.Sprintf("%s/publish/%s/status", packagePath(packageID), url.PathEscape(publishID))
	var st PublishStatusResponse
	code, raw, err := c.doJSON(http.MethodGet, path, nil, nil, &st)
	if err != nil {
		return nil, code, fmt.Errorf("publish status decode: %w body=%s", err, string(raw))
	}
	if code != http.StatusOK {
		return nil, code, fmt.Errorf("publish status: status %d: %s", code, string(raw))
	}
	return &st, code, nil
}
