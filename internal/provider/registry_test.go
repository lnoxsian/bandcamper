package provider

import (
	"context"
	"net/url"
	"testing"
)

type mockProvider struct {
	name   string
	domain string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) CanHandle(u *url.URL) bool {
	return u != nil && u.Hostname() == m.domain
}

func (m *mockProvider) Resolve(ctx context.Context, u *url.URL) ([]*Release, error) {
	return []*Release{
		{
			Provider: m.name,
			Artist:   "Mock Artist",
			Album:    "Mock Album",
			Tracks: []Track{
				{Number: 1, Title: "Track One", StreamURL: "http://example.com/audio.mp3"},
			},
		},
	}, nil
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	p1 := &mockProvider{name: "alpha", domain: "alpha.example.com"}
	p2 := &mockProvider{name: "beta", domain: "beta.example.com"}

	reg.Register(p1)
	reg.Register(p2)

	if len(reg.Providers()) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(reg.Providers()))
	}

	// Auto-resolve by URL
	u1, _ := url.Parse("https://alpha.example.com/item/1")
	found, err := reg.ResolveProvider(u1, "")
	if err != nil {
		t.Fatalf("unexpected error resolving alpha: %v", err)
	}
	if found.Name() != "alpha" {
		t.Errorf("expected alpha, got %s", found.Name())
	}

	// Explicit provider override
	u2, _ := url.Parse("https://unknown.com/item")
	foundExplicit, err := reg.ResolveProvider(u2, "beta")
	if err != nil {
		t.Fatalf("unexpected error with explicit beta: %v", err)
	}
	if foundExplicit.Name() != "beta" {
		t.Errorf("expected beta, got %s", foundExplicit.Name())
	}

	// Unknown explicit provider
	_, err = reg.ResolveProvider(u2, "gamma")
	if err == nil {
		t.Errorf("expected error for unknown provider gamma")
	}

	// No provider matches
	_, err = reg.ResolveProvider(u2, "")
	if err == nil {
		t.Errorf("expected error when no provider matches URL")
	}
}
