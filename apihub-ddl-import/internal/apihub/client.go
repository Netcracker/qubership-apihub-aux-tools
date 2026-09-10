// Package apihub is the APIHUB REST client of the DDL import tool. The
// transport core mirrors apihub-portal-package-copy/internal/apihub (tools do
// not import across modules); the DDL-specific methods are new.
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

// VerboseHTTP enables extra HTTP diagnostics when HTTPDebug is set (from --debug).
var VerboseHTTP bool

// HTTPDebug receives formatted lines (typically stderr); nil disables logging.
var HTTPDebug func(format string, args ...any)

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// New returns an API client. If insecureSkipTLSVerify is true, TLS server
// certificates are not verified; use only with private CAs or troubleshooting.
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
	r, err := http.NewRequest(method, c.BaseURL+path, body)
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
	msg := "(no body)"
	if len(body) > 0 {
		s := strings.TrimSpace(string(body))
		if len(s) > 2000 {
			s = s[:2000] + "… [truncated]"
		}
		msg = s
	}
	HTTPDebug("HTTP %-6s %-80s ← %d  %s", method, path, status, msg)
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
			HTTPDebug("HTTP %s %s → error: %v", method, path, err)
		}
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
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

// APIError is a non-2xx APIHUB response with the backend's error envelope
// ({"status":..,"code":"8403","message":..}) parsed when present.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
	Raw        string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("apihub: status %d code %s: %s", e.HTTPStatus, e.Code, e.Message)
	}
	return fmt.Sprintf("apihub: status %d: %s", e.HTTPStatus, e.Raw)
}

func apiError(status int, body []byte) *APIError {
	e := &APIError{HTTPStatus: status, Raw: strings.TrimSpace(string(body))}
	var env struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &env) == nil {
		e.Code = env.Code
		e.Message = env.Message
	}
	return e
}

func pkgPath(packageID string) string {
	return "/api/v2/packages/" + url.PathEscape(packageID)
}

func versionPath(packageID, version string) string {
	return "/api/v3/packages/" + url.PathEscape(packageID) + "/versions/" + url.PathEscape(version)
}

func ddlPath(packageID, version, tail string) string {
	return "/api/v1/packages/" + url.PathEscape(packageID) + "/versions/" + url.PathEscape(version) + "/ddl" + tail
}

// VersionInfo is the subset of the version content the tool needs.
type VersionInfo struct {
	PackageID       string `json:"packageId"`
	Version         string `json:"version"`
	Status          string `json:"status"`
	PreviousVersion string `json:"previousVersion,omitempty"`
}

// GetVersion returns the version info, or found=false on 404.
func (c *Client) GetVersion(packageID, version string) (*VersionInfo, bool, error) {
	var v VersionInfo
	code, raw, err := c.doJSON(http.MethodGet, versionPath(packageID, version), nil, nil, &v)
	if err != nil {
		return nil, false, fmt.Errorf("get version %s@%s: %w", packageID, version, err)
	}
	switch code {
	case http.StatusOK:
		return &v, true, nil
	case http.StatusNotFound:
		return nil, false, nil
	default:
		return nil, false, apiError(code, raw)
	}
}

type publishAccepted struct {
	PublishId string `json:"publishId"`
}

// PublishVersion posts a multipart publish (config JSON + sources zip).
// sync=true means HTTP 204: nothing to build, no polling needed.
func (c *Client) PublishVersion(packageID string, sourcesZip, configJSON []byte) (publishID string, sync bool, err error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormField("config")
	if err != nil {
		return "", false, err
	}
	if _, err := fw.Write(configJSON); err != nil {
		return "", false, err
	}
	part, err := w.CreateFormFile("sources", "sources.zip")
	if err != nil {
		return "", false, err
	}
	if _, err := part.Write(sourcesZip); err != nil {
		return "", false, err
	}
	_ = w.WriteField("resolveRefs", "true")
	_ = w.WriteField("resolveConflicts", "true")
	if err := w.Close(); err != nil {
		return "", false, err
	}
	r, err := c.req(http.MethodPost, pkgPath(packageID)+"/publish", &b, map[string]string{"Content-Type": w.FormDataContentType()})
	if err != nil {
		return "", false, err
	}
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	httpDebugLog(http.MethodPost, pkgPath(packageID)+"/publish (multipart)", resp.StatusCode, raw)
	switch resp.StatusCode {
	case http.StatusNoContent:
		return "", true, nil
	case http.StatusAccepted:
		var acc publishAccepted
		if err := json.Unmarshal(raw, &acc); err != nil {
			return "", false, fmt.Errorf("decode publish response: %w body=%s", err, string(raw))
		}
		return acc.PublishId, false, nil
	default:
		return "", false, apiError(resp.StatusCode, raw)
	}
}

// PublishStatus is the build status of a publish operation.
type PublishStatus struct {
	PublishId string `json:"publishId"`
	Status    string `json:"status"` // none | running | complete | error
	Message   string `json:"message"`
}

func (c *Client) GetPublishStatus(packageID, publishID string) (*PublishStatus, error) {
	var st PublishStatus
	code, raw, err := c.doJSON(http.MethodGet,
		pkgPath(packageID)+"/publish/"+url.PathEscape(publishID)+"/status", nil, nil, &st)
	if err != nil {
		return nil, fmt.Errorf("publish status: %w", err)
	}
	if code != http.StatusOK {
		return nil, apiError(code, raw)
	}
	return &st, nil
}

// DdlEntity is one published DDL entity (table or view).
type DdlEntity struct {
	DdlEntityId string `json:"ddlEntityId"`
	Kind        string `json:"kind"`
	SchemaName  string `json:"schemaName"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DocumentId  string `json:"documentId"`
	PackageRef  string `json:"packageRef,omitempty"`
}

type ddlEntityListView struct {
	Entities []DdlEntity `json:"entities"`
}

// ListDdlEntities returns all DDL entities of the version (paginating with the
// backend's limit=100 maximum).
func (c *Client) ListDdlEntities(packageID, version string) ([]DdlEntity, error) {
	const limit = 100
	var all []DdlEntity
	for offset := 0; ; offset += limit {
		q := url.Values{}
		q.Set("limit", strconv.Itoa(limit))
		q.Set("offset", strconv.Itoa(offset))
		var page ddlEntityListView
		code, raw, err := c.doJSON(http.MethodGet, ddlPath(packageID, version, "/entities")+"?"+q.Encode(), nil, nil, &page)
		if err != nil {
			return nil, fmt.Errorf("list ddl entities: %w", err)
		}
		if code != http.StatusOK {
			return nil, apiError(code, raw)
		}
		all = append(all, page.Entities...)
		if len(page.Entities) < limit {
			return all, nil
		}
	}
}

// DdlChange is one change of a DDL entity between two versions.
type DdlChange struct {
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

type ddlEntityChangesView struct {
	Changes []DdlChange `json:"changes"`
}

// GetDdlEntityChanges returns the per-change list for one entity against the
// previous version.
func (c *Client) GetDdlEntityChanges(packageID, version, ddlEntityID, previousVersion, previousVersionPackageID string) ([]DdlChange, error) {
	q := url.Values{}
	if previousVersion != "" {
		q.Set("previousVersion", previousVersion)
	}
	if previousVersionPackageID != "" {
		q.Set("previousVersionPackageId", previousVersionPackageID)
	}
	p := ddlPath(packageID, version, "/entities/"+url.PathEscape(ddlEntityID)+"/changes")
	if enc := q.Encode(); enc != "" {
		p += "?" + enc
	}
	var view ddlEntityChangesView
	code, raw, err := c.doJSON(http.MethodGet, p, nil, nil, &view)
	if err != nil {
		return nil, fmt.Errorf("ddl entity changes: %w", err)
	}
	if code != http.StatusOK {
		return nil, apiError(code, raw)
	}
	return view.Changes, nil
}

// ExportDdlEntities downloads the entities xlsx report.
func (c *Client) ExportDdlEntities(packageID, version string) ([]byte, error) {
	return c.getBinary(ddlPath(packageID, version, "/export/entities"))
}

// ExportDdlChanges downloads the changes xlsx report against previousVersion.
func (c *Client) ExportDdlChanges(packageID, version, previousVersion, previousVersionPackageID string) ([]byte, error) {
	q := url.Values{}
	q.Set("previousVersion", previousVersion)
	if previousVersionPackageID != "" {
		q.Set("previousVersionPackageId", previousVersionPackageID)
	}
	return c.getBinary(ddlPath(packageID, version, "/export/changes") + "?" + q.Encode())
}

func (c *Client) getBinary(path string) ([]byte, error) {
	r, err := c.req(http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		httpDebugLog(http.MethodGet, path, resp.StatusCode, b)
		return nil, apiError(resp.StatusCode, b)
	}
	if VerboseHTTP && HTTPDebug != nil {
		HTTPDebug("HTTP %-6s %-80s ← %d  (binary payload %d bytes)", http.MethodGet, path, resp.StatusCode, len(b))
	}
	return b, nil
}

// DdlGroupTable references a DDL entity to include in a group. PackageId and
// Version stay empty for entities of the group's own version (dashboards only).
type DdlGroupTable struct {
	PackageId   string `json:"packageId,omitempty"`
	Version     string `json:"version,omitempty"`
	DdlEntityId string `json:"ddlEntityId"`
}

type CreateDdlGroupReq struct {
	GroupName   string          `json:"groupName"`
	Description string          `json:"description,omitempty"`
	Tables      []DdlGroupTable `json:"tables"`
}

type UpdateDdlGroupReq struct {
	GroupName   *string          `json:"groupName,omitempty"`
	Description *string          `json:"description,omitempty"`
	Tables      *[]DdlGroupTable `json:"tables,omitempty"`
}

// CreateDdlGroup creates a DDL table group in the version.
func (c *Client) CreateDdlGroup(packageID, version string, body CreateDdlGroupReq) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	code, raw, err := c.doJSON(http.MethodPost, ddlPath(packageID, version, "/groups"), bytes.NewReader(buf), nil, nil)
	if err != nil {
		return fmt.Errorf("create ddl group %s: %w", body.GroupName, err)
	}
	if code != http.StatusCreated && code != http.StatusOK {
		return apiError(code, raw)
	}
	return nil
}

// UpdateDdlGroup patches a DDL table group (tables are replaced wholesale).
func (c *Client) UpdateDdlGroup(packageID, version, groupName string, body UpdateDdlGroupReq) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	code, raw, err := c.doJSON(http.MethodPatch,
		ddlPath(packageID, version, "/groups/"+url.PathEscape(groupName)), bytes.NewReader(buf), nil, nil)
	if err != nil {
		return fmt.Errorf("update ddl group %s: %w", groupName, err)
	}
	if code < 200 || code >= 300 {
		return apiError(code, raw)
	}
	return nil
}
