package provider

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	// ErrNoProviderMatches indicates no registered provider recognizes the URL.
	ErrNoProviderMatches = errors.New("no registered provider can handle this URL")
	// ErrProviderNotFound indicates a provider explicitly requested by name was not found.
	ErrProviderNotFound = errors.New("requested provider not found")
)

// Registry manages registered providers and selects the appropriate provider for a given URL.
type Registry struct {
	providers []Provider
}

// NewRegistry initializes an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make([]Provider, 0),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	if p == nil {
		return
	}
	r.providers = append(r.providers, p)
}

// Providers returns all registered providers.
func (r *Registry) Providers() []Provider {
	res := make([]Provider, len(r.providers))
	copy(res, r.providers)
	return res
}

// ResolveProvider finds the provider that handles the URL, or matches explicitName if non-empty.
func (r *Registry) ResolveProvider(u *url.URL, explicitName string) (Provider, error) {
	if explicitName != "" {
		for _, p := range r.providers {
			if strings.EqualFold(p.Name(), explicitName) {
				return p, nil
			}
		}
		return nil, fmt.Errorf("%w: %q", ErrProviderNotFound, explicitName)
	}

	for _, p := range r.providers {
		if p.CanHandle(u) {
			return p, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNoProviderMatches, u.String())
}
