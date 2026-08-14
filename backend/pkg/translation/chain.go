package translation

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chorus/messenger/pkg/logutil"
)

// ChainProvider wraps multiple Provider instances and tries them in sequence.
// It implements the Provider interface so it's a drop-in replacement anywhere
// a single provider is used.
//
// Failure handling:
//   - Any error falls through to the next provider.
//   - Rate limits (429), exhausted quotas (403 quota), auth failures (401/403)
//     and transient server errors (5xx) additionally put the provider into a
//     cooldown window: it is skipped entirely on subsequent requests until the
//     window expires, so a rate-limited provider is not hammered repeatedly.
//   - Providers missing required config (e.g. an empty API key) are reported
//     via Health() at startup and skipped fast at runtime.
type ChainProvider struct {
	providers []Provider
	mu        sync.Mutex
	cooldowns map[string]time.Time
}

// NewChainProvider creates a chain from the given providers. Providers are
// tried in order — the first successful response wins.
func NewChainProvider(providers []Provider) *ChainProvider {
	return &ChainProvider{
		providers: providers,
		cooldowns: make(map[string]time.Time),
	}
}

// Translate tries each provider in sequence. If all fail, returns the last error.
func (c *ChainProvider) Translate(ctx context.Context, req TranslateRequest) (TranslateResponse, error) {
	var lastErr error
	tried := 0
	for i, p := range c.providers {
		name := p.Name()
		if coolDownUntil, cooling := c.coolDownFor(name); cooling {
			logutil.Debugf("[Chain] skipping %s (cooling down until %s)", name, coolDownUntil.Format(time.RFC3339))
			continue
		}
		tried++
		logutil.Infof("[Chain] trying provider %d/%d: %s (text_len=%d target=%s)",
			i+1, len(c.providers), name, len(req.Text), req.TargetLang)
		start := time.Now()
		resp, err := p.Translate(ctx, req)
		logutil.Duration("Chain", start, name)
		if err == nil {
			resp.Provider = name
			logutil.Infof("[Chain] provider %d/%d %s succeeded", i+1, len(c.providers), name)
			return resp, nil
		}
		lastErr = err
		c.markFailure(name, err)
		logutil.Warnf("[Chain] provider %d/%d %s failed: %v — trying next",
			i+1, len(c.providers), name, err)
	}
	if tried == 0 {
		return TranslateResponse{},
			fmt.Errorf("all %d providers in cooldown or unavailable: %w", len(c.providers), lastErr)
	}
	return TranslateResponse{},
		fmt.Errorf("all %d providers exhausted: %w", len(c.providers), lastErr)
}

// markFailure records a provider failure and applies a cooldown when the error
// class (rate limit, quota, auth, 5xx) warrants pausing the provider.
func (c *ChainProvider) markFailure(name string, err error) {
	cooldown := CooldownFor(err)
	if cooldown <= 0 {
		return
	}
	c.mu.Lock()
	c.cooldowns[name] = time.Now().Add(cooldown)
	c.mu.Unlock()
	logutil.Warnf("[Chain] %s cooling down for %v (%v)", name, cooldown, err)
}

// DetectLanguage tries each provider that implements LanguageDetector in
// chain order and returns the first successful ISO 639-1 code. Providers that
// are cooling down are skipped, mirroring the Translate behavior.
func (c *ChainProvider) DetectLanguage(ctx context.Context, text string) (string, error) {
	var lastErr error
	for _, p := range c.providers {
		detector, ok := p.(LanguageDetector)
		if !ok {
			continue
		}
		if coolDownUntil, cooling := c.coolDownFor(p.Name()); cooling {
			logutil.Debugf("[Chain] skipping %s for detect (cooling down until %s)", p.Name(), coolDownUntil.Format(time.RFC3339))
			continue
		}
		code, err := detector.DetectLanguage(ctx, text)
		if err == nil && code != "" {
			logutil.Infof("[Chain] language detected by %s: %s", p.Name(), code)
			return code, nil
		}
		lastErr = err
		c.markFailure(p.Name(), err)
		logutil.Warnf("[Chain] %s language detection failed: %v — trying next", p.Name(), err)
	}
	if lastErr == nil {
		return "", fmt.Errorf("no provider in the chain supports language detection")
	}
	return "", fmt.Errorf("language detection failed across chain: %w", lastErr)
}

// coolDownFor returns the provider's cooldown expiry, if it is currently paused.
func (c *ChainProvider) coolDownFor(name string) (time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	until, ok := c.cooldowns[name]
	if !ok {
		return time.Time{}, false
	}
	if time.Now().After(until) {
		delete(c.cooldowns, name)
		return time.Time{}, false
	}
	return until, true
}

// Healthy reports which providers are ready to handle requests. A provider is
// considered misconfigured (unhealthy) when it needs an API key but has none.
// This is used by startup checks to warn about missing keys before traffic.
func (c *ChainProvider) Healthy() []ProviderHealth {
	var out []ProviderHealth
	for _, p := range c.providers {
		h := ProviderHealth{Name: p.Name(), Ready: true}
		if oc, ok := p.(interface{ NeedsKey() bool }); ok {
			if oc.NeedsKey() {
				h.Ready = false
				h.Reason = "missing API key (see startup config warnings)"
			}
		}
		out = append(out, h)
	}
	return out
}

// ProviderHealth describes the readiness of a single provider in the chain.
type ProviderHealth struct {
	Name   string
	Ready  bool
	Reason string
}

// Providers exposes the underlying providers in chain order. Useful for
// startup probes and health reporting.
func (c *ChainProvider) Providers() []Provider {
	return c.providers
}

// Pingable is implemented by providers that support a lightweight readiness
// probe (currently the local/offline providers).
type Pingable interface {
	Ping(ctx context.Context) error
}

// ModelEnsurer is implemented by providers that can auto-install a missing
// model (currently Ollama via /api/pull). Used so the offline provider
// self-heals at startup instead of only warning.
type ModelEnsurer interface {
	EnsureModel(ctx context.Context) error
}

// ProbeResult describes the outcome of a startup readiness probe.
type ProbeResult struct {
	Name      string
	Reachable bool
	Err       error
}

// Probe checks reachability of the given providers. Only providers that
// implement Pingable (local/offline backends) are probed; cloud providers are
// skipped and reported as reachable=false with Err=nil. Results are advisory —
// a failed probe means the provider will be skipped at runtime, not that the
// server cannot start.
func Probe(ctx context.Context, providers []Provider) []ProbeResult {
	var results []ProbeResult
	for _, p := range providers {
		pingable, ok := p.(Pingable)
		if !ok {
			// Cloud provider: no probe. Skip (not part of the offline guarantee).
			continue
		}
		res := ProbeResult{Name: p.Name()}
		if err := pingable.Ping(ctx); err != nil {
			res.Reachable = false
			res.Err = err
		} else {
			res.Reachable = true
		}
		results = append(results, res)
	}
	return results
}

// Name returns the full chain description for logging/metrics.
func (c *ChainProvider) Name() string {
	names := make([]string, len(c.providers))
	for i, p := range c.providers {
		names[i] = p.Name()
	}
	return "chain(" + strings.Join(names, " → ") + ")"
}
