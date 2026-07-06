package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaGenerateResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// NLLB API request/response types
type NLLBRequest struct {
	Text   string `json:"text"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type NLLBResponse struct {
	TranslatedText string `json:"translatedText"`
}

type TranslationQueueJob struct {
	MessageID    string   `json:"messageId"`
	Text         string   `json:"text"`
	TargetLangs  []string `json:"targetLangs"`
}

type TranslationService struct {
	redis        *redis.Client
	ollamaURL    string
	ollamaModel  string
	nllbURL      string
	httpClient   *http.Client
	queueEnabled bool
	ctx          context.Context
}

func NewTranslationService(nllbURL, ollamaURL, ollamaModel string, redis *redis.Client) *TranslationService {
	return &TranslationService{
		redis:        redis,
		ollamaURL:    ollamaURL,
		ollamaModel:  ollamaModel,
		nllbURL:      nllbURL,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
		queueEnabled: true,
		ctx:          context.Background(),
	}
}

func (s *TranslationService) Translate(text, targetLang string) (string, error) {
	return s.TranslateQuick(text, targetLang, "auto")
}

// TranslateQuick translates text using NLLB (Phase 1 — fast, 200+ languages).
// Retries up to 3 times with backoff to handle NLLB container startup delay.
func (s *TranslationService) TranslateQuick(text, targetLang, sourceLang string) (string, error) {
	cacheKey := fmt.Sprintf("translation:nllb:%s:%s", targetLang, text)
	if s.redis != nil {
		cached, err := s.redis.Get(s.ctx, cacheKey).Result()
		if err == nil && cached != "" {
			return cached, nil
		}
	}
	if s.nllbURL == "" {
		return "", errors.New("NLLB URL not configured")
	}

	// Convert ISO codes to FLORES-200 codes for NLLB
	sourceFlores := isoToFlores(sourceLang)
	targetFlores := isoToFlores(targetLang)

	reqBody := NLLBRequest{
		Text:   text,
		Source: sourceFlores,
		Target: targetFlores,
	}
	body, _ := json.Marshal(reqBody)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}

		req, _ := http.NewRequestWithContext(s.ctx, "POST", s.nllbURL+"/translate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("nllb call (attempt %d): %w", attempt+1, err)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			var nllbResp NLLBResponse
			json.NewDecoder(resp.Body).Decode(&nllbResp)
			resp.Body.Close()
			if nllbResp.TranslatedText != "" {
				if s.redis != nil {
					s.redis.Set(s.ctx, cacheKey, nllbResp.TranslatedText, 24*time.Hour)
				}
				return nllbResp.TranslatedText, nil
			}
			resp.Body.Close()
		} else {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("nllb status %d (attempt %d): %s", resp.StatusCode, attempt+1, string(respBody))
		}
	}
	return "", lastErr
}

func (s *TranslationService) TranslateWithOllama(text, targetLang string) (string, error) {
	if s.ollamaURL == "" {
		return "", errors.New("Ollama URL not configured")
	}
	langName := languageCodeToName(targetLang)
	prompt := fmt.Sprintf(
		"Translate the following text to %s. Return ONLY the translated text, preserving all original line breaks and formatting. Do not add any explanation or prefix.\n\n%s",
		langName, text,
	)
	reqBody := OllamaGenerateRequest{Model: s.ollamaModel, Prompt: prompt, Stream: false}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(s.ctx, "POST", s.ollamaURL+"/api/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama call: %w", err)
	}
	defer resp.Body.Close()
	var ollamaResp OllamaGenerateResponse
	json.NewDecoder(resp.Body).Decode(&ollamaResp)
	result := strings.TrimSpace(ollamaResp.Response)
	if len(result) >= 2 {
		if (result[0] == '"' && result[len(result)-1] == '"') ||
			(result[0] == '\'' && result[len(result)-1] == '\'') {
			result = result[1 : len(result)-1]
		}
	}
	if result == "" {
		return "", errors.New("ollama empty response")
	}
	if s.redis != nil {
		s.redis.Set(s.ctx, fmt.Sprintf("translation:ollama:%s:%s", targetLang, text), result, 24*time.Hour)
	}
	return result, nil
}

func (s *TranslationService) TranslateMultiple(text string, targetLangs []string) (map[string]string, error) {
	translations := make(map[string]string)
	for _, lang := range targetLangs {
		trans, err := s.Translate(text, lang)
		if err != nil {
			continue
		}
		translations[lang] = trans
	}
	return translations, nil
}

func (s *TranslationService) EnqueueOllamaTranslation(messageID, text string, targetLangs []string) error {
	if s.redis == nil {
		return errors.New("redis not available")
	}
	job := TranslationQueueJob{MessageID: messageID, Text: text, TargetLangs: targetLangs}
	data, _ := json.Marshal(job)
	return s.redis.LPush(s.ctx, "translation:ollama:queue", data).Err()
}

func (s *TranslationService) ProcessOllamaQueue(onComplete func(messageID string, translations map[string]string)) {
	if s.redis == nil {
		return
	}
	for {
		result, err := s.redis.BRPop(s.ctx, 30*time.Second, "translation:ollama:queue").Result()
		if err != nil {
			continue
		}
		if len(result) < 2 {
			continue
		}
		var job TranslationQueueJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			continue
		}
		translations := make(map[string]string)
		for _, lang := range job.TargetLangs {
			trans, err := s.TranslateWithOllama(job.Text, lang)
			if err != nil {
				fmt.Printf("[OllamaQueue] failed %s: %v\n", lang, err)
				continue
			}
			translations[lang] = trans
		}
		if len(translations) > 0 {
			onComplete(job.MessageID, translations)
		}
	}
}

// isoToFlores converts ISO 639-1 codes to FLORES-200 codes used by NLLB.
func isoToFlores(code string) string {
	if code == "" || code == "auto" {
		return "eng_Latn"
	}
	code = strings.ToLower(strings.Split(code, "-")[0])
	m := map[string]string{
		"en": "eng_Latn", "es": "spa_Latn", "fr": "fra_Latn", "de": "deu_Latn",
		"it": "ita_Latn", "pt": "por_Latn", "ja": "jpn_Jpan", "ko": "kor_Hang",
		"zh": "zho_Hans", "ar": "arb_Arab", "nl": "nld_Latn", "pl": "pol_Latn",
		"ru": "rus_Cyrl", "sv": "swe_Latn",
		"af": "afr_Latn", "bg": "bul_Cyrl", "bn": "ben_Beng", "bs": "bos_Latn",
		"ca": "cat_Latn", "cs": "ces_Latn", "cy": "cym_Latn", "da": "dan_Latn",
		"el": "ell_Grek", "et": "est_Latn", "fa": "pes_Arab", "fi": "fin_Latn",
		"ga": "gle_Latn", "gl": "glg_Latn", "gu": "guj_Gujr", "ha": "hau_Latn",
		"he": "heb_Hebr", "hi": "hin_Deva", "hr": "hrv_Latn", "hu": "hun_Latn",
		"id": "ind_Latn", "ig": "ibo_Latn", "is": "isl_Latn", "kk": "kaz_Cyrl",
		"km": "khm_Khmr", "kn": "kan_Knda", "ky": "kir_Cyrl", "lo": "lao_Laoo",
		"lt": "lit_Latn", "lv": "lav_Latn", "mg": "mlg_Latn", "mk": "mkd_Cyrl",
		"ml": "mal_Mlym", "mn": "mon_Cyrl", "mr": "mar_Deva", "ms": "msa_Latn",
		"mt": "mlt_Latn", "my": "mya_Mymr", "ne": "npi_Deva", "no": "nob_Latn",
		"pa": "pan_Guru", "ps": "pbt_Arab", "ro": "ron_Latn", "rw": "kin_Latn",
		"si": "sin_Sinh", "sk": "slk_Latn", "sl": "slv_Latn", "so": "som_Latn",
		"sq": "sqi_Latn", "sr": "srp_Cyrl", "sw": "swh_Latn", "ta": "tam_Taml",
		"te": "tel_Telu", "tg": "tgk_Cyrl", "th": "tha_Thai", "tk": "tuk_Latn",
		"tr": "tur_Latn", "uk": "ukr_Cyrl", "ur": "urd_Arab", "uz": "uzn_Latn",
		"vi": "vie_Latn", "xh": "xho_Latn", "yo": "yor_Latn", "zu": "zul_Latn",
	}
	if v, ok := m[code]; ok {
		return v
	}
	return code + "_Latn"
}

func languageCodeToName(code string) string {
	m := map[string]string{
		"en": "English", "es": "Spanish", "fr": "French", "de": "German",
		"it": "Italian", "pt": "Portuguese", "ja": "Japanese", "ko": "Korean",
		"zh": "Chinese", "ar": "Arabic", "nl": "Dutch", "pl": "Polish",
		"ru": "Russian", "sv": "Swedish", "af": "Afrikaans", "bg": "Bulgarian",
		"bn": "Bengali", "bs": "Bosnian", "ca": "Catalan", "cs": "Czech",
		"cy": "Welsh", "da": "Danish", "el": "Greek", "et": "Estonian",
		"fa": "Persian", "fi": "Finnish", "ga": "Irish", "gl": "Galician",
		"gu": "Gujarati", "ha": "Hausa", "he": "Hebrew", "hi": "Hindi",
		"hr": "Croatian", "hu": "Hungarian", "id": "Indonesian", "ig": "Igbo",
		"is": "Icelandic", "kk": "Kazakh", "km": "Khmer", "kn": "Kannada",
		"ky": "Kyrgyz", "lo": "Lao", "lt": "Lithuanian", "lv": "Latvian",
		"mg": "Malagasy", "mk": "Macedonian", "ml": "Malayalam", "mn": "Mongolian",
		"mr": "Marathi", "ms": "Malay", "mt": "Maltese", "my": "Burmese",
		"ne": "Nepali", "no": "Norwegian", "pa": "Punjabi", "ps": "Pashto",
		"ro": "Romanian", "rw": "Kinyarwanda", "si": "Sinhala", "sk": "Slovak",
		"sl": "Slovenian", "so": "Somali", "sq": "Albanian", "sr": "Serbian",
		"sw": "Swahili", "ta": "Tamil", "te": "Telugu", "tg": "Tajik",
		"th": "Thai", "tk": "Turkmen", "tr": "Turkish", "uk": "Ukrainian",
		"ur": "Urdu", "uz": "Uzbek", "vi": "Vietnamese", "xh": "Xhosa",
		"yo": "Yoruba", "zu": "Zulu",
	}
	if v, ok := m[code]; ok {
		return v
	}
	return code
}