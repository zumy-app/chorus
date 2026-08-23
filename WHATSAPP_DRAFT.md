# WhatsApp draft — chorus.talk Phase 1 consolidation (copy-paste)

**Option A — Detailed (for main group):**

Hi all — quick update from today's chorus.talk sync (Aug 23) 👇

*Minutes — key points agreed:*
• Provide toggles to turn translation/grammar on/off
• Keep a learned word-bank — don't re-translate known words like "hola"/"hi"; one-tap add to bank + highlight new words for practice
• Dashboard "Your Learning Path": show words learned / sentences understood per month (real metrics, not mocks)
• Quality loop: store all translations + grammar, re-score with a *different* model, critique prompts, track KPIs (accuracy/latency/cost) via Arize Phoenix (local, offline+realtime)
• Writing goals: draft in target language with AI feedback/gap-fill before sending; AI scenario role-play (restaurant, date, etc.)
• Chat Language Settings: keep only *your* language dropdown (remove other person's)
• Teacher vetting: assessment + live recording + certs → manual video review by language expert (Daniella for Spanish, 1 expert/lang) → later train AI
• Security: Mailu is on same prod host → isolate
• Infra: create `dev` env, Raju owns CI/CD + quality gates (functional + e2e + Phoenix eval + load smoke) before auto-promote dev→prod

*Decision: Phase 0 + Phase 1 MERGED*
→ Only one milestone now: **Phase 1 Release**. Goal for Phase 1 = working translation + grammar + mobile app (Expo) + core NFRs/security + structured learning activities & tracking. All ex-Phase 0 bugs (home link, admin back, premium copy, emoji picker, contacts/invites, first/last name, avatar, error handling) are now Phase 1 P0 (launch-blocking).

*Backlog update:*
• Pulled 60 existing issues. 10 ex-Phase 0 issues will be moved to Phase 1 P0.
• 14 new issues created from the notes above (feature controls, word-bank optimization, Learning Path metrics, highlight+practice, Phoenix pipeline, writing assistant, scenario role-play, simplify language modal, mail isolation, dev env+gates, teacher vetting doc, housekeeping).
• Scripts ready: `scripts/create_phase1_issues.ps1` / `.sh` + `scripts/move_phase0_to_phase1.ps1` — dry-run passes.

*Heads up:* GitHub PAT in `.env` (`GITHUB_CHORUS_ISSUES_PAT`) is returning 401 Bad credentials, so I couldn't push via API. Please rotate a Fine-grained PAT for `zumy-app/chorus` (Issues: Read & write) and re-run, or I can create manually. Docs are committed locally: `REQUIREMENTS.md`, `PHASE_1_IMPLEMENTATION_PLAN.md`, `BACKLOG_REFINEMENT_2026-08-23.md`.

Owners: Mobile P0 Kushagra, Learning Gayatri, Dev env Raju, Security/Infra batchu. Next: P0 in 1–2 wks, then translation hardening + Learn page wiring.

Let me know if any of the above needs correction 🙏

---

**Option B — Short (if you want 1 paragraph):**

Chorus.talk sync 23 Aug: agreed Phase 0+1 → single Phase 1 (P0=launch blockers). New scope: translation toggles, word-bank dedup, Learning Path metrics (words/sentences per month), highlight+practice, Phoenix quality eval, AI writing help + scenario role-play, simplify language settings, isolate Mailu, dev env with gates (Raju). 14 new issues + 10 moved — scripts ready but GH PAT is expired (401) so needs rotate to push. Details in REQUIREMENTS.md / backlog doc. Owners: Kushagra (mobile), Gayatri (learning), Raju (CI/CD).
