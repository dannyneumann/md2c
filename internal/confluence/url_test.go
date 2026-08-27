package confluence

import "testing"

func TestNormalizeAPIBase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"https://acme.atlassian.net", "https://acme.atlassian.net/wiki/rest/api"},
		{"https://acme.atlassian.net/", "https://acme.atlassian.net/wiki/rest/api"},
		{"https://acme.atlassian.net/wiki", "https://acme.atlassian.net/wiki/rest/api"},
		{"https://acme.atlassian.net/wiki/rest/api", "https://acme.atlassian.net/wiki/rest/api"},
		{"https://acme.atlassian.net/wiki/rest/api/", "https://acme.atlassian.net/wiki/rest/api"},
		{"https://confluence.example.com", "https://confluence.example.com/rest/api"},
		{"https://confluence.example.com/wiki", "https://confluence.example.com/wiki/rest/api"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeAPIBase(tt.in); got != tt.want {
			t.Errorf("NormalizeAPIBase(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestSplitPath(t *testing.T) {
	t.Parallel()
	got := SplitPath(" /Engineering/ Onboarding / ")
	if len(got) != 2 || got[0] != "Engineering" || got[1] != "Onboarding" {
		t.Fatalf("got %#v", got)
	}
	if len(SplitPath("   ")) != 0 {
		t.Fatal("empty path should yield no segments")
	}
}
