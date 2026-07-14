package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultFastTimeout   = 5 * time.Second
	defaultTutorTimeout  = 10 * time.Second
	defaultOllamaPath   = "/api/chat"
	defaultTemperature  = 0.1
	defaultTopP         = 0.9
	defaultNumPredict   = 128
	defaultNumThread    = 3
)

type Engine struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

type ChatRequest struct {
	Model    string         `json:"model"`
	Messages []ChatMessage  `json:"messages"`
	Stream   bool           `json:"stream"`
	Format   string         `json:"format,omitempty"`
	Options  map[string]any `json:"options,omitempty"`
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Message ChatResponseMessage `json:"message"`
}

type ChatResponseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TutorJSONReport struct {
	Language    string              `json:"language"`
	Corrections []GrammarCorrection `json:"corrections"`
	Vocabulary  []VocabularyItem    `json:"vocabulary"`
	Notes       string              `json:"notes"`
}

type GrammarCorrection struct {
	Original    string `json:"original"`
	Corrected   string `json:"corrected"`
	Explanation string `json:"explanation"`
}

type VocabularyItem struct {
	Word        string `json:"word"`
	Translation string `json:"translation"`
	Context     string `json:"context"`
}

func NewEngine(baseURL, model string) *Engine {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5:1.5b-instruct"
	}
	return &Engine{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      model,
		HTTPClient: &http.Client{Timeout: defaultTutorTimeout},
	}
}

func (e *Engine) ExecuteFastTranslation(text, srcLang, tgtLang string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultFastTimeout)
	defer cancel()

	if strings.TrimSpace(text) == "" {
		return "", nil
	}
	if e == nil || e.BaseURL == "" {
		return "", fmt.Errorf("ollama engine not configured")
	}

	prompt := fmt.Sprintf("You are a deterministic translation engine. Translate the user text to %s. Return only the translated text.\n\nSource language: %s\nTarget language: %s\nText: %s", tgtLang, srcLang, tgtLang, text)
	payload := ChatRequest{
		Model: e.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: "Translate the text with no preamble or explanation. Return only the translated text."},
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Options: map[string]any{
			"temperature": 0.1,
			"top_p":       0.9,
			"num_predict": 64,
			"num_thread":  3,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal translation payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+defaultOllamaPath, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create translation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("translation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", fmt.Errorf("translation request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var response struct {
		Message ChatResponseMessage `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode translation response: %w", err)
	}

	translated := strings.TrimSpace(response.Message.Content)
	translated = strings.Trim(translated, "`\n\r\t")
	if translated == "" {
		return "", fmt.Errorf("translation response empty")
	}
	return translated, nil
}

func (e *Engine) ProcessTutorAnalysis(content, srcLang, tgtLang string) (TutorJSONReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTutorTimeout)
	defer cancel()

	if strings.TrimSpace(content) == "" {
		return TutorJSONReport{}, nil
	}
	if e == nil || e.BaseURL == "" {
		return TutorJSONReport{}, fmt.Errorf("ollama engine not configured")
	}

	prompt := fmt.Sprintf(`You are a strict grammar tutor. Return only a JSON object matching this schema and nothing else.

Schema:
{
  "language": "string",
  "corrections": [
    {"original": "text", "corrected": "text", "explanation": "short explanation"}
  ],
  "vocabulary": [
    {"word": "word", "translation": "translation", "context": "short context"}
  ],
  "notes": "short note"
}

Rules:
- Never use markdown, code fences, or prose outside JSON.
- If the input is already correct, return an empty corrections array.
- Keep the output strict JSON.

Example input:
"I am go to school yesterday."

Example output:
{"language":"en","corrections":[{"original":"I am go to school yesterday.","corrected":"I went to school yesterday.","explanation":"Corrected the verb tense and grammar."}],"vocabulary":[{"word":"school","translation":"escuela","context":"a place for learning"}],"notes":"The sentence needed a tense correction."}

Input language: %s
Target language: %s
Content: %s`, srcLang, tgtLang, content)

	payload := ChatRequest{
		Model: e.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: "Return only strict JSON. Do not add markdown or prose."},
			{Role: "user", Content: prompt},
		},
		Stream: false,
		Format: "json",
		Options: map[string]any{
			"temperature": 0.1,
			"top_p":       0.9,
			"num_predict": defaultNumPredict,
			"num_thread":  defaultNumThread,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return TutorJSONReport{}, fmt.Errorf("marshal tutor payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+defaultOllamaPath, bytes.NewReader(body))
	if err != nil {
		return TutorJSONReport{}, fmt.Errorf("create tutor request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.HTTPClient.Do(req)
	if err != nil {
		return TutorJSONReport{}, fmt.Errorf("tutor request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return TutorJSONReport{}, fmt.Errorf("tutor request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	var response struct {
		Message ChatResponseMessage `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return TutorJSONReport{}, fmt.Errorf("decode tutor response: %w", err)
	}

	contentBody := strings.TrimSpace(response.Message.Content)
	contentBody = strings.Trim(contentBody, "`\n\r\t")
	var report TutorJSONReport
	if err := json.Unmarshal([]byte(contentBody), &report); err != nil {
		return TutorJSONReport{}, fmt.Errorf("parse tutor json: %w", err)
	}
	return report, nil
}
