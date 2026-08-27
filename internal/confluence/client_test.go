package confluence

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type stored struct {
	title   string
	parent  string
	body    string
	version int
	id      string
}

func TestFindPageMissing(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/content" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.URL.Query().Get("spaceKey") != "DEV" || r.URL.Query().Get("title") != "Onboarding" {
			t.Errorf("query %s", r.URL.RawQuery)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "me" || pass != "token" {
			t.Errorf("basic auth %q %q", user, pass)
		}
		_, _ = w.Write([]byte(`{"results":[],"size":0}`))
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv)
	page, err := c.FindPage(context.Background(), "DEV", "Onboarding")
	if err != nil {
		t.Fatal(err)
	}
	if page != nil {
		t.Fatalf("expected nil page, got %+v", page)
	}
}

func TestBearerAuth(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer pat-token" {
			t.Errorf("authorization %q", r.Header.Get("Authorization"))
		}
		if _, _, ok := r.BasicAuth(); ok {
			t.Errorf("unexpected basic auth")
		}
		_, _ = w.Write([]byte(`{"results":[],"size":0}`))
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv)
	c.Auth = "bearer"
	c.Token = "pat-token"
	_, err := c.FindPage(context.Background(), "DEV", "Onboarding")
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublishCreatesHierarchyThenUpdates(t *testing.T) {
	t.Parallel()

	pages := map[string]*stored{}
	nextID := 1

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/content":
			title := r.URL.Query().Get("title")
			p, ok := pages[title]
			if !ok {
				_, _ = w.Write([]byte(`{"results":[],"size":0}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"results": []map[string]any{pageJSON(p)},
				"size":    1,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/content":
			raw, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Errorf("create payload: %v", err)
			}
			title, _ := payload["title"].(string)
			body := extractStorage(payload)
			parent := ""
			if anc, ok := payload["ancestors"].([]any); ok && len(anc) > 0 {
				if m, ok := anc[0].(map[string]any); ok {
					parent, _ = m["id"].(string)
				}
			}
			id := nextID
			nextID++
			p := &stored{
				title:   title,
				parent:  parent,
				body:    body,
				version: 1,
				id:      strconv.Itoa(id),
			}
			pages[title] = p
			_ = json.NewEncoder(w).Encode(pageJSON(p))
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/rest/api/content/"):
			raw, _ := io.ReadAll(r.Body)
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Errorf("update payload: %v", err)
			}
			title, _ := payload["title"].(string)
			p := pages[title]
			if p == nil {
				http.NotFound(w, r)
				return
			}
			p.body = extractStorage(payload)
			p.version++
			_ = json.NewEncoder(w).Encode(pageJSON(p))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv)
	ctx := context.Background()

	created, err := c.Publish(ctx, "DEV", "Engineering/Onboarding", "<p>hello</p>")
	if err != nil {
		t.Fatal(err)
	}
	if created.Title != "Onboarding" {
		t.Fatalf("title %q", created.Title)
	}
	parent := pages["Engineering"]
	leaf := pages["Onboarding"]
	if parent == nil || leaf == nil {
		t.Fatalf("pages not created: %+v", pages)
	}
	if parent.body != "<p></p>" {
		t.Fatalf("parent body %q", parent.body)
	}
	if leaf.parent != parent.id {
		t.Fatalf("leaf parent %q want %q", leaf.parent, parent.id)
	}
	if leaf.body != "<p>hello</p>" {
		t.Fatalf("leaf body %q", leaf.body)
	}

	updated, err := c.Publish(ctx, "DEV", "Engineering/Onboarding", "<p>updated</p>")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version.Number != 2 {
		t.Fatalf("version %d", updated.Version.Number)
	}
	if pages["Onboarding"].body != "<p>updated</p>" {
		t.Fatalf("updated body %q", pages["Onboarding"].body)
	}
}

func TestAuthError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"not authorized"}`))
	}))
	t.Cleanup(srv.Close)

	c := testClient(srv)
	_, err := c.FindPage(context.Background(), "DEV", "X")
	if err == nil || !strings.Contains(err.Error(), "MD2C_USER") {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestPageNumericID(t *testing.T) {
	t.Parallel()
	var p Page
	if err := json.Unmarshal([]byte(`{"id":12345,"type":"page","title":"T","version":{"number":1}}`), &p); err != nil {
		t.Fatal(err)
	}
	if p.ID != "12345" {
		t.Fatalf("id %q", p.ID)
	}
}

func TestPublishEmptyPath(t *testing.T) {
	t.Parallel()
	c := New("https://example.com", "u", "t")
	_, err := c.Publish(context.Background(), "DEV", "  /  ", "<p></p>")
	if err == nil {
		t.Fatal("expected error")
	}
}

func testClient(srv *httptest.Server) *Client {
	c := New(srv.URL, "me", "token")
	c.HTTPClient = srv.Client()
	return c
}

func pageJSON(p *stored) map[string]any {
	return map[string]any{
		"id":    p.id,
		"type":  "page",
		"title": p.title,
		"space": map[string]string{"key": "DEV"},
		"version": map[string]int{
			"number": p.version,
		},
		"_links": map[string]string{
			"base":  "https://acme.atlassian.net/wiki",
			"webui": "/spaces/DEV/pages/" + p.id + "/" + p.title,
		},
	}
}

func extractStorage(payload map[string]any) string {
	body, _ := payload["body"].(map[string]any)
	storage, _ := body["storage"].(map[string]any)
	value, _ := storage["value"].(string)
	return value
}
