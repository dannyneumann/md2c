package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 60 * time.Second

// Client talks to the Confluence REST API (Cloud and Server/DC v1).
type Client struct {
	BaseURL    string
	User       string
	Token      string
	Auth       string
	HTTPClient *http.Client
	UserAgent  string
}

// Page is a Confluence content page.
type Page struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	Space   Space   `json:"space"`
	Version Version `json:"version"`
	Links   Links   `json:"_links"`
}

// Space identifies a Confluence space.
type Space struct {
	Key string `json:"key"`
}

// Version is the optimistic-lock version of a page.
type Version struct {
	Number int `json:"number"`
}

// Links holds Confluence navigation links.
type Links struct {
	WebUI string `json:"webui"`
	Base  string `json:"base"`
}

type listResponse struct {
	Results []Page `json:"results"`
	Size    int    `json:"size"`
}

type apiError struct {
	Message string `json:"message"`
}

// New constructs a Client with a timeout-enabled HTTP client.
func New(baseURL, user, token string) *Client {
	return &Client{
		BaseURL: NormalizeAPIBase(baseURL),
		User:    user,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: defaultTimeout,
		},
		UserAgent: "md2c",
		Auth:      "basic",
	}
}

// WebURL returns the browser URL for a page when Confluence provided links.
func (p Page) WebURL() string {
	if p.Links.Base != "" && p.Links.WebUI != "" {
		return strings.TrimRight(p.Links.Base, "/") + p.Links.WebUI
	}
	return ""
}

// FindPage returns the page with the given title in a space, or (nil, nil) if missing.
func (c *Client) FindPage(ctx context.Context, space, title string) (*Page, error) {
	q := url.Values{}
	q.Set("spaceKey", space)
	q.Set("title", title)
	q.Set("expand", "version,ancestors,space,_links")
	q.Set("limit", "1")

	var list listResponse
	if err := c.do(ctx, http.MethodGet, "/content?"+q.Encode(), nil, &list); err != nil {
		return nil, err
	}
	if len(list.Results) == 0 {
		return nil, nil
	}
	page := list.Results[0]
	return &page, nil
}

// CreatePage creates a page, optionally under a parent.
func (c *Client) CreatePage(ctx context.Context, space, title, parentID, storage string) (*Page, error) {
	body := map[string]any{
		"type":  "page",
		"title": title,
		"space": map[string]string{"key": space},
		"body": map[string]any{
			"storage": map[string]string{
				"value":          storage,
				"representation": "storage",
			},
		},
	}
	if parentID != "" {
		body["ancestors"] = []map[string]string{{"id": parentID}}
	}

	var page Page
	if err := c.do(ctx, http.MethodPost, "/content", body, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// UpdatePage replaces the storage body of an existing page.
func (c *Client) UpdatePage(ctx context.Context, page *Page, storage string) (*Page, error) {
	body := map[string]any{
		"id":    page.ID,
		"type":  "page",
		"title": page.Title,
		"space": map[string]string{"key": page.Space.Key},
		"body": map[string]any{
			"storage": map[string]string{
				"value":          storage,
				"representation": "storage",
			},
		},
		"version": map[string]any{
			"number":  page.Version.Number + 1,
			"message": "updated by md2c",
		},
	}

	var updated Page
	if err := c.do(ctx, http.MethodPut, "/content/"+url.PathEscape(page.ID), body, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Publish creates or updates the page at space/path. Intermediate path
// segments become parent pages when they do not exist. The last segment is
// the page title whose body is replaced with storage.
func (c *Client) Publish(ctx context.Context, space, pagePath, storage string) (*Page, error) {
	segments := SplitPath(pagePath)
	if len(segments) == 0 {
		return nil, fmt.Errorf("path is empty")
	}

	parentID := ""
	for i, title := range segments {
		leaf := i == len(segments)-1
		page, err := c.FindPage(ctx, space, title)
		if err != nil {
			return nil, err
		}
		if page == nil {
			body := "<p></p>"
			if leaf {
				body = storage
			}
			created, err := c.CreatePage(ctx, space, title, parentID, body)
			if err != nil {
				return nil, fmt.Errorf("create page %q: %w", title, err)
			}
			if leaf {
				return created, nil
			}
			parentID = created.ID
			continue
		}
		if leaf {
			updated, err := c.UpdatePage(ctx, page, storage)
			if err != nil {
				return nil, fmt.Errorf("update page %q: %w", title, err)
			}
			return updated, nil
		}
		parentID = page.ID
	}
	return nil, fmt.Errorf("publish: no leaf page")
}

// SplitPath splits a Confluence page path on '/' and trims empty segments.
func SplitPath(pagePath string) []string {
	raw := strings.Split(pagePath, "/")
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (c *Client) do(ctx context.Context, method, path string, payload any, dest any) error {
	if c.BaseURL == "" {
		return fmt.Errorf("confluence base URL is not set")
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: defaultTimeout}
	}

	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Atlassian-Token", "no-check")
	if strings.EqualFold(c.Auth, "bearer") {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	} else {
		req.SetBasicAuth(c.User, c.Token)
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s", method, path, formatAPIError(resp.StatusCode, raw))
	}
	if dest == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func formatAPIError(status int, raw []byte) string {
	msg := strings.TrimSpace(string(raw))
	var ae apiError
	if json.Unmarshal(raw, &ae) == nil && ae.Message != "" {
		msg = ae.Message
	}
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		if strings.Contains(msg, "Standardauthentifizierung") || strings.Contains(strings.ToLower(msg), "basic authentication") {
			return fmt.Sprintf("HTTP %d: Basic-Auth ist auf dieser Instanz aus. In md2c.conf MD2C_AUTH=bearer setzen und ein Confluence-Zugriffstoken verwenden: %s", status, msg)
		}
		return fmt.Sprintf("HTTP %d (check MD2C_USER / MD2C_TOKEN / MD2C_AUTH): %s", status, msg)
	default:
		return fmt.Sprintf("HTTP %d: %s", status, msg)
	}
}

// UnmarshalJSON accepts Confluence ids encoded as string or number.
func (p *Page) UnmarshalJSON(data []byte) error {
	type alias Page
	aux := &struct {
		ID json.RawMessage `json:"id"`
		*alias
	}{
		alias: (*alias)(p),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.ID) == 0 || string(aux.ID) == "null" {
		return nil
	}
	var asString string
	if err := json.Unmarshal(aux.ID, &asString); err == nil {
		p.ID = asString
		return nil
	}
	var asNumber json.Number
	if err := json.Unmarshal(aux.ID, &asNumber); err != nil {
		return fmt.Errorf("page id: %w", err)
	}
	p.ID = asNumber.String()
	return nil
}
