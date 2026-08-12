package config

import (
	"testing"
)

func setEnv(t *testing.T, pairs map[string]string) {
	t.Helper()
	for k, v := range pairs {
		t.Setenv(k, v)
	}
}

func TestLoadNewStyleProviderKeys(t *testing.T) {
	setEnv(t, map[string]string{
		"TRANSLATION_FALLBACK_ORDER":      "openrouter,nvidia,opencode,ollama,libretranslate",
		"GRAMMAR_FALLBACK_ORDER":          "opencode,openrouter,nvidia,ollama",
		"PROVIDER_OPENROUTER_URL":         "https://openrouter.ai/api/v1",
		"PROVIDER_OPENROUTER_KEY":         "sk-openrouter",
		"PROVIDER_OPENCODE_URL":           "https://opencode.ai/zen/go/v1",
		"PROVIDER_OPENCODE_KEY":           "sk-opencode",
		"PROVIDER_NVIDIA_URL":             "https://integrate.api.nvidia.com/v1",
		"PROVIDER_NVIDIA_KEY":             "nvapi-test",
		"PROVIDER_OLLAMA_URL":             "http://ollama:11434",
		"PROVIDER_OLLAMA_TIMEOUT":         "60",
		"PROVIDER_LIBRETRANSLATE_URL":     "http://localhost:5000",
		"PROVIDER_LIBRETRANSLATE_TIMEOUT": "10",
		"MODEL_TRANSLATION_OPENROUTER":    "google/gemma-4-26b-a4b-it:free",
		"MODEL_GRAMMAR_OPENROUTER":        "meta/llama-3.1-8b-instruct",
		"MODEL_TRANSLATION_OPENCODE":      "deepseek-v4-flash",
		"MODEL_GRAMMAR_OPENCODE":          "deepseek-v4-flash",
		"MODEL_TRANSLATION_NVIDIA":        "meta/llama-3.1-70b-instruct",
		"MODEL_GRAMMAR_NVIDIA":            "meta/llama-3.1-8b-instruct",
		"MODEL_TRANSLATION_OLLAMA":        "qwen2.5:3b",
		"MODEL_GRAMMAR_OLLAMA":            "qwen2.5:1.5b",
	})

	cfg := Load()

	wantOrder := []string{"openrouter", "nvidia", "opencode", "ollama", "libretranslate"}
	if len(cfg.TranslationProviderOrder) != len(wantOrder) {
		t.Fatalf("TranslationProviderOrder = %v, want %v", cfg.TranslationProviderOrder, wantOrder)
	}
	for i, name := range wantOrder {
		if cfg.TranslationProviderOrder[i] != name {
			t.Fatalf("TranslationProviderOrder[%d] = %q, want %q", i, cfg.TranslationProviderOrder[i], name)
		}
	}

	wantGrammar := []string{"opencode", "openrouter", "nvidia", "ollama"}
	if len(cfg.GrammarProviderOrder) != len(wantGrammar) {
		t.Fatalf("GrammarProviderOrder = %v, want %v", cfg.GrammarProviderOrder, wantGrammar)
	}

	or := cfg.Providers["openrouter"]
	if or.URL != "https://openrouter.ai/api/v1" || or.Key != "sk-openrouter" || or.URL == "" || or.Key == "" {
		t.Fatalf("openrouter def = %+v", or)
	}
	if or.EffectiveType() != "openrouter" {
		t.Fatalf("openrouter EffectiveType = %q", or.EffectiveType())
	}
	if or.TranslationModel() != "google/gemma-4-26b-a4b-it:free" {
		t.Fatalf("openrouter TranslationModel = %q", or.TranslationModel())
	}
	if or.GrammarModel() != "meta/llama-3.1-8b-instruct" {
		t.Fatalf("openrouter GrammarModel = %q", or.GrammarModel())
	}

	oc := cfg.Providers["opencode"]
	if oc.URL != "https://opencode.ai/zen/go/v1" || oc.Key != "sk-opencode" {
		t.Fatalf("opencode def = %+v", oc)
	}
	if oc.EffectiveType() != "opencode" {
		t.Fatalf("opencode EffectiveType = %q", oc.EffectiveType())
	}
	if oc.TranslationModel() != "deepseek-v4-flash" || oc.GrammarModel() != "deepseek-v4-flash" {
		t.Fatalf("opencode models = t:%q g:%q", oc.TranslationModel(), oc.GrammarModel())
	}

	ol := cfg.Providers["ollama"]
	if ol.URL != "http://ollama:11434" || ol.Timeout != 60 {
		t.Fatalf("ollama def = %+v", ol)
	}
	if ol.EffectiveType() != "ollama" {
		t.Fatalf("ollama EffectiveType = %q", ol.EffectiveType())
	}
	if ol.TranslationModel() != "qwen2.5:3b" || ol.GrammarModel() != "qwen2.5:1.5b" {
		t.Fatalf("ollama models = t:%q g:%q", ol.TranslationModel(), ol.GrammarModel())
	}

	lr := cfg.Providers["libretranslate"]
	if lr.URL != "http://localhost:5000" || lr.Timeout != 10 {
		t.Fatalf("libretranslate def = %+v", lr)
	}
	if lr.EffectiveType() != "libretranslate" {
		t.Fatalf("libretranslate EffectiveType = %q", lr.EffectiveType())
	}
}

func TestLoadLegacyProviderKeys(t *testing.T) {
	setEnv(t, map[string]string{
		"TRANSLATION_PROVIDER_ORDER":      "openrouter,nvidia",
		"GRAMMAR_ANALYSIS_PROVIDER_ORDER": "openrouter",
		"PROVIDER_OPENROUTER_TYPE":        "openrouter",
		"PROVIDER_OPENROUTER_API_URL":     "https://openrouter.ai/api/v1",
		"PROVIDER_OPENROUTER_API_KEY":     "sk-legacy",
		"PROVIDER_OPENROUTER_MODEL":       "google/gemma-4-26b-a4b-it:free",
		"PROVIDER_NVIDIA_TYPE":            "nvidia",
		"PROVIDER_NVIDIA_API_URL":         "https://integrate.api.nvidia.com/v1",
		"PROVIDER_NVIDIA_API_KEY":         "nvapi-legacy",
		"PROVIDER_NVIDIA_MODEL":           "meta/llama-3.1-70b-instruct",
		"PROVIDER_OLLAMA_LOCAL_TYPE":      "ollama",
		"PROVIDER_OLLAMA_LOCAL_API_URL":   "http://ollama:11434",
		"PROVIDER_OLLAMA_LOCAL_MODEL":     "qwen2.5:3b",
		"PROVIDER_OLLAMA_LOCAL_TIMEOUT":   "600",
	})

	cfg := Load()

	if len(cfg.TranslationProviderOrder) != 2 || cfg.TranslationProviderOrder[0] != "openrouter" || cfg.TranslationProviderOrder[1] != "nvidia" {
		t.Fatalf("TranslationProviderOrder = %v", cfg.TranslationProviderOrder)
	}
	if len(cfg.GrammarProviderOrder) != 1 || cfg.GrammarProviderOrder[0] != "openrouter" {
		t.Fatalf("GrammarProviderOrder = %v", cfg.GrammarProviderOrder)
	}

	or := cfg.Providers["openrouter"]
	if or.URL != "https://openrouter.ai/api/v1" || or.Key != "sk-legacy" {
		t.Fatalf("legacy openrouter def = %+v", or)
	}
	if or.EffectiveType() != "openrouter" {
		t.Fatalf("legacy openrouter EffectiveType = %q", or.EffectiveType())
	}
	if or.TranslationModel() != "google/gemma-4-26b-a4b-it:free" || or.GrammarModel() != "google/gemma-4-26b-a4b-it:free" {
		t.Fatalf("legacy openrouter models = t:%q g:%q", or.TranslationModel(), or.GrammarModel())
	}

	ol, ok := cfg.Providers["ollama_local"]
	if !ok {
		ol, ok = cfg.Providers["ollamalocal"]
	}
	if !ok {
		t.Fatalf("ollama_local not found in providers: %v", cfg.Providers)
	}
	if ol.URL != "http://ollama:11434" || ol.Timeout != 600 {
		t.Fatalf("legacy ollama def = %+v", ol)
	}
}

func TestEnvKeyFor(t *testing.T) {
	if got := EnvKeyFor("openrouter", "KEY"); got != "PROVIDER_OPENROUTER_KEY" {
		t.Fatalf("EnvKeyFor(openrouter,KEY) = %q", got)
	}
	if got := EnvKeyFor("openrouter", "URL"); got != "PROVIDER_OPENROUTER_URL" {
		t.Fatalf("EnvKeyFor(openrouter,URL) = %q", got)
	}
	if got := ModelTranslationKeyFor("nvidia"); got != "MODEL_TRANSLATION_NVIDIA" {
		t.Fatalf("ModelTranslationKeyFor(nvidia) = %q", got)
	}
	if got := ModelGrammarKeyFor("ollama"); got != "MODEL_GRAMMAR_OLLAMA" {
		t.Fatalf("ModelGrammarKeyFor(ollama) = %q", got)
	}
}

func TestProviderTypeDerivation(t *testing.T) {
	cases := map[string]string{
		"openrouter":     "openrouter",
		"opencode":       "opencode",
		"nvidia":         "nvidia",
		"ollama":         "ollama",
		"libretranslate": "libretranslate",
	}
	for name, want := range cases {
		def := ProviderDef{Name: name}
		if got := def.EffectiveType(); got != want {
			t.Errorf("EffectiveType(%s) = %q, want %q", name, got, want)
		}
	}
}
