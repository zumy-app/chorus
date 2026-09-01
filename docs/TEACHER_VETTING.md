# Teacher vetting process (11.4) — assessment → recording → expert video review

Source: NFR-27, BACKLOG_REFINEMENT 2026-08-23 §6.11, REQUIREMENTS.md NFR-27, PHASE_1_IMPLEMENTATION_PLAN §10.
Owner: batchu · Expert lead: Daniella (Spanish) + one expert per language · Status: doc-only in Phase 1, execution Phase 2.

Marketplace profiles, ratings, and class sign-up (epic #53) are gated on this vetting. No teacher is surfaced until `vetting_status = approved`.

## Pipeline

```
apply → basic assessment → live recording → certificates → expert video-call review → decision → (Phase 2) listing
```

Each stage is manual in Phase 1; Phase 2 adds the harness below to assist reviewers. Later the rubric dataset trains an AI evaluator.

| Stage | Artifact | Owner | SLA | Gate |
|-------|----------|-------|-----|------|
| 1 Apply | `teacher_applications` row: bio, languages, rate, video URL | Applicant | — | required fields + word limit 280/1000 |
| 2 Assessment | Written placement (CEFR B2+ in teaching language, 20 Q) + `assessment_score` | System / reviewer | auto-scored, <48h review | score ≥ 70% |
| 3 Live recording | 2–3 min unscripted language demo (prompt below) | Applicant | upload ≤ 100 MB, mp4/webm | file + duration check |
| 4 Certificates | Upload certs (PDF/JPG), type + issuer + year | Applicant | — | checklist §3 |
| 5 Expert video-call | 20 min structured interview via Chorus video call | Expert (e.g. Daniella for ES) | scheduled ≤ 5 days | rubric score |
| 6 Decision | `approved` / `needs_work` / `rejected` + notes | Expert + batchu | ≤ 24h after call | marketplace gate |

State machine: `pending → assessment_passed → recording_uploaded → certs_verified → review_scheduled → reviewed → {approved, rejected, needs_work}`. `needs_work` may resubmit once after 14 days.

## Rubric

Scored 1–5 per criterion; weighted. Pass requires weighted avg ≥ 3.6 and no criterion ≤ 2, plus at least one 4+ in pedagogy.

| # | Criterion | Weight | 5 = excellent, 3 = adequate, 1 = fail |
|---|-----------|--------|----------------------------------------|
| 1 | Pronunciation & intelligibility (target language) | 0.22 | Native-like / clear, minor accent OK |
| 2 | Fluency & coherence | 0.18 | Sustained, well-linked discourse |
| 3 | Pedagogical clarity | 0.22 | Explains concepts, checks comprehension |
| 4 | Engagement & rapport | 0.14 | Warm, adaptive, encourages output |
| 5 | Accuracy (grammar/vocab) | 0.14 | CEFR C1 accuracy, self-corrects |
| 6 | Professional conduct | 0.10 | Punctual, tech-ready, boundaries |

Score formula: `weighted = Σ score_i * weight_i`. See `backend/internal/services/teacher_vetting.go:ScoreRubric`.

Hard fails: hate/harassment, misrepresented credentials, unreadable recording → immediate `rejected`.

## Recording prompt library

Applicant picks one per submission; prompts are CEFR-aligned and must be answered in the teaching language without reading.

- A2: Introduce yourself and your hometown (1 min).
- B1: Explain how to order coffee at a market (1.5 min).
- B1: Describe your last trip and what you learned (2 min).
- B2: Teach the difference between `por` vs `para` with two examples (2 min).
- B2 (ES): Explain `subjunctive` with a contrast to indicative (2 min).
- C1: Role-play: student is at a restaurant and is nervous; coach them (2 min).
- Any: 30s spontaneous: "What would you do if a student freezes?" (unscripted).

Recording checks (harness): duration 90–210s, audio RMS > threshold, no screen-read detection (speech rate variance).

## Certificate checklist

Accepted types: `teaching_degree`, `language_certificate` (DELE, CELTA, etc.), `other`. Each row: `type`, `issuer`, `year`, `file_url`, `verified` (reviewer tick). At least one teaching or C1+ language cert required for `approved`; otherwise `needs_work` with path to obtain.

Storage: `teacher_certificates` table, files in `deploy/storage` (prod: S3), scanned for size/mime.

## Expert video-call flow

Pre-call (5 min): reviewer reads application + recording + certs, pre-scores rubric draft.
Call (20 min):
- 0–2 min intro + consent (recorded if applicant consents; otherwise notes only).
- 2–8 min language demo: expert gives a micro-prompt (e.g., "teach me past tense"), applicant teaches.
- 8–14 min pedagogy: scenario drill + correction; expert notes comprehension checks.
- 14–18 min Q&A: availability, rate, conduct, boundaries.
- 18–20 min wrap + next steps.

Post-call: expert submits `RubricScore` + free-text notes + decision within 24h. Second reviewer required if weighted 3.4–3.7.

Tooling: uses existing Chorus WebRTC video call (dual-view, captions). Calendar invite via email; stipend noted per session in ops sheet.

## Expert roster

| Language | Expert | Status | Calendar |
|----------|--------|--------|----------|
| Spanish (ES) | Daniella | confirmed in writing (2026-08-23 sync) | daniella@chorus.talk — via batchu |
| English (EN) | TBD — recruit 1 | open — target before Phase 2 execution | — |
| Other | one per language (FR, DE, etc.) | recruit as demand | — |

Recruitment: language-teaching background, CEFR C1+, prior reviewer experience. Added to `teacher_experts` roster table.

## Marketplace gating (Phase 2 execution)

- `teacher_profiles vetting_status` is source of truth; search/listing queries filter `approved` only.
- Pending/rejected profiles return 404 to students; owner sees banner + resubmit CTA.
- Ratings and booking are disabled until approved.

## AI training plan (future)

Dataset: rubric rows + recording transcripts + call notes (consented) → `teacher_vetting_evals`. Split: train 80 / held-out 20. Model: fine-tuned evaluator that predicts per-criterion scores from transcript + cert metadata; expert remains tie-breaker until accuracy ≥ 0.85 vs held-out. No auto-approval before human sign-off.

## Verification harness

Pure-Go harness at `backend/internal/services/teacher_vetting.go` (`AssessmentResult`, `ValidateApplication`, `ValidateRecording`, `ValidateCertificates`, `ScoreRubric`, `Decide`):

```bash
go vet ./...
go test ./internal/services -run TestTeacherVetting -count=1 -v
bash deploy/ci/verify-teacher-vetting.sh
```

`deploy/ci/verify-teacher-vetting.sh` asserts: doc exists, required headings present, harness tests pass, rubric weights sum to 1.0, and `Daniella` roster entry present.

## Metrics & alerts (Phase 2)

- `chorus_teacher_applications_total{status}` and `chorus_teacher_vetting_duration_hours`.
- Alert if `review_scheduled` age > 5 days.

## Appeals

`rejected` may appeal once within 14 days with new evidence; routed to a different expert. `needs_work` may resubmit after addressing notes.

## Acceptance (Phase 1)

- [x] `docs/TEACHER_VETTING.md` merged.
- [x] Expert Daniella confirmed in writing.
- [x] Harness `go test -run TestTeacherVetting` green; `verify-teacher-vetting.sh` green.
