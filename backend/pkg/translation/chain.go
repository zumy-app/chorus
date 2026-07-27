package translation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chorus/messenger/pkg/logutil"
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
		logutil.Infof("[Chain] trying provider %d/%d: %s (text_len=%d target=%s)",
			i+1, len(c.providers), p.Name(), len(req.Text), req.TargetLang)
		start := time.Now()
		resp, err := p.Translate(ctx, req)
		logutil.Duration("Chain", start, p.Name())
		if err == nil {
			resp.Provider = fmt.Sprintf("%s", p.Name())
			logutil.Infof("[Chain] provider %d/%d %s succeeded", i+1, len(c.providers), p.Name())
			return resp, nil
		}
		lastErr = err
		logutil.Warnf("[Chain] provider %d/%d %s failed: %v — trying next",
			i+1, len(c.providers), p.Name(), err)
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
