package models

import "time"

type MinedItem struct {
	ID                  string    `json:"id" db:"id"`
	UserID              string    `json:"userId" db:"user_id"`
	JobID               string    `json:"jobId,omitempty" db:"job_id"`
	ChatID              string    `json:"chatId,omitempty" db:"chat_id"`
	MessageID           string    `json:"messageId,omitempty" db:"message_id"`
	SourceType          string    `json:"sourceType" db:"source_type"`
	SurfaceText         string    `json:"surfaceText" db:"surface_text"`
	Lemma               string    `json:"lemma" db:"lemma"`
	NormalizedText      string    `json:"normalizedText" db:"normalized_text"`
	Language            string    `json:"language" db:"language"`
	PartOfSpeech        string    `json:"partOfSpeech" db:"part_of_speech"`
	Translation         string    `json:"translation" db:"translation"`
	Definition          string    `json:"definition" db:"definition"`
	ContextSentence     string    `json:"contextSentence" db:"context_sentence"`
	TextSpan            any       `json:"textSpan,omitempty" db:"text_span"`
	CEFRLevel           string    `json:"cefrLevel,omitempty" db:"cefr_level"`
	Confidence          float64   `json:"confidence" db:"confidence"`
	TeachabilityScore   float64   `json:"teachabilityScore" db:"teachability_score"`
	IsChunk             bool      `json:"isChunk" db:"is_chunk"`
	IsProperNoun        bool      `json:"isProperNoun" db:"is_proper_noun"`
	GrammarTags         []string  `json:"grammarTags,omitempty" db:"grammar_tags"`
	CurriculumLexicalID string    `json:"curriculumLexicalItemId,omitempty" db:"curriculum_lexical_item_id"`
	CurriculumUnitID    string    `json:"curriculumUnitId,omitempty" db:"curriculum_unit_id"`
	RouteStatus         string    `json:"routeStatus" db:"route_status"`
	Status              string    `json:"status" db:"status"`
	RouteReason         string    `json:"routeReason" db:"route_reason"`
	CreatedAt           time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time `json:"updatedAt" db:"updated_at"`
}
