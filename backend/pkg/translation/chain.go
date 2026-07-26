package translation

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// ChainProvider wraps multiple Provider instances and tries them in sequence.
// It implements the Provider interface so it's a drop-in replacement anywhere
// a single provider is used. On failure (including 429 rate limits), it falls
// through to the next provider in the chain.
type ChainProvider struct {
	providers []Provider
}

// NewChainProvider creates a chain from the given providers. Providers are
// tried in order — the first successful response wins.
func NewChainProvider(providers []Provider) *ChainProvider {
	return &ChainProvider{providers: providers}
}

// Translate tries each provider in sequence. If all fail, returns the last error.
func (c *ChainProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	var lastErr error
	for i, p := range c.providers {
		resp, err := p.Translate(ctx, req)
		if err == nil {
			resp.Provider = fmt.Sprintf("%s", p.Name())
			return resp, nil
		}
		lastErr = err
		log.Printf("[Translate] provider %d (%s) failed: %.80s — trying next",
			i, p.Name(), err.Error())
	}
	return TranslateResponse{},
		fmt.Errorf("all %d providers exhausted: %w", len(c.providers), lastErr)
}

// Name returns the full chain description for logging/metrics.
func (c *ChainProvider) Name() string {
	names := make([]string, len(c.providers))
	for i, p := range c.providers {
		names[i] = p.Name()
	}
	return "chain(" + strings.Join(names, " → ") + ")"
}
