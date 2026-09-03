# REQUIREMENTS SLICE — Home v2 (S-HOME-01..04) — BA Trace

> **Authority:** `wireframes/chorus_home_desktop_v2/code.html:134` (newest home — canonical), `wireframes/chorus_home_desktop_v2/code.html:1` header/navigation, `frontend/src/pages/Landing.tsx:47` (current stale `STRINGS` — to be purged), `mobile/src/screens/LandingScreen.tsx:96` (current stale hero — to be purged), `REQUIREMENTS_MASTER.md:1` (machine-readable backlog), `REQUIREMENTS.md:1` (narrative), `docs/WIREFRAME_TRACE.md:27`, `docs/CREWAI_GAP_CLOSURE_PLAN.md:58`, `backend/cmd/server/main.go:467` (`GET /health`), `backend/internal/observability/health.go:5`
> **BA Owner:** analyst (`crew/roles.py:22` — owns requirements traceability matrix, maps every wireframe → requirement id → code, gap list is source of truth, QA cannot pass until trace green)
> **Common Contract:** `crew/roles.py:97` — mobile-first / web parity (NFR-22), Go+Gin + Postgres (source of truth) + Redis (cache/pubsub/registry only), Vite React web, Expo RN primary, wireframes ARE spec
> **Generated:** 2026-09-03 | **Status:** BA FINAL — rebuild decision locked, impl blocked until QA writes failing tests first (`docs/CREWAI_GAP_CLOSURE_PLAN.md:39` Stage 2 red gate)

---

## 0. Decision — Rebuild to `chorus_home_desktop_v2` (not freeze v1)

**Decision: REBUILD both surfaces to `wireframes/chorus_home_desktop_v2/code.html:134` as single canonical Home. Freeze of `chorus_home/code.html:200` (`Break Language Barriers` cafe/waitlist) is REJECTED.**

**Rationale (recorded per `docs/CREWAI_GAP_CLOSURE_PLAN.md:60`):** `chorus_home_desktop_v2` is the newest Figma, the only version that contains the language-learning positioning (`Communication is Learning` rather than generic translation utility), the 4-card ecosystem that teases the Teacher Marketplace (required for monetization `REQUIREMENTS_MASTER.md:145` / `REQUIREMENTS.md:139` P1–P4), and the brain-hero + mockup assets needed for Phase 2+ upsell. `Landing.tsx:47` / `LandingScreen.tsx:96` currently implement the **stale v1** copy and must be fully replaced — no hybrid.

**Stale copy to purge (verify zero hits after impl):**
```
grep -rn "Break Language Barriers" frontend/src/pages/Landing.tsx mobile/src/screens/LandingScreen.tsx
grep -rn "Connect Globally" frontend/ mobile/
grep -rn "Real-time messaging with instant translation" frontend/ mobile/
grep -rn "Powerful Features for Global Communication" frontend/ mobile/
grep -rn "Instant Translation" frontend/src/pages/Landing.tsx mobile/src/screens/LandingScreen.tsx
grep -rn "Grammar Analysis" frontend/src/pages/Landing.tsx mobile/src/screens/LandingScreen.tsx  # v1 label
grep -rn "Vocabulary Builder" frontend/ mobile/
grep -rn "Group Chats" frontend/ mobile/          # v1 6-card include
grep -rn "Privacy First" frontend/ mobile/
grep -rn "How Chorus Works" frontend/ mobile/      # v1 how-it-works
grep -rn "Available in {count} languages" frontend/ mobile/  # v1 badge
grep -rn "Powerful Features" frontend/ mobile/
grep -rn "79\.90" frontend/ mobile/                # v1 yearly $79.90 — must become $7.99/mo
grep -rn "Ready to Break Language Barriers" frontend/ mobile/
grep -rn "enterprise" -i frontend/src/pages/Landing.tsx mobile/src/screens/LandingScreen.tsx  # v1 3-tier (Free/Premium/Enterprise) removed
```
After impl, the only pricing tiers rendered are **Free $0** + **Premium $7.99/mo** (v2 `code.html:222`). Enterprise card is deleted.

**Canonical v2 content contract (every bullet must render on both surfaces):**

| # | Section | Wireframe ref | Copy / asset contract |
|---|---|---|---|
| 1 | **TopNavBar** | `code.html:115-132` | Logo, nav `Features`→`#features` `:122`, `Pricing`→`#pricing` `:123`, `About Us`→`#about` `:124`, CTA `Get Started` `:127` sticky `backdrop-blur` |
| 2 | **Hero — brain** | `code.html:134-163` | `h1` `Communication is Learning.` `+ Redefining how we acquire language.` `:138` + `p` `Bridging the gap between messaging apps and learning platforms… making communication and learning the exact same function.` `:141` + visual `img alt="Brain Neural Pathways"` `:157` + CTAs `Start Your Journey` `:145` (primary) / `Watch Demo` + `play_circle` `:148` (secondary) |
| 3 | **Bridging (Problem/Solution)** | `code.html:164-173` | `h2` `Bridging Messaging and Learning. The Best of Both Worlds.` `:167-168` + `p` `Why choose between a messenger like WhatsApp and a learning tool like Duolingo? Chorus combines them…` `:168-171` centered `max-w-4xl` |
| 4 | **Ecosystem — 4 cards + mockup** | `code.html:174-220` | `h2` `A Complete Language Ecosystem` `:178` + `p` `Everything you need to go from basic phrases to true fluency.` `:179` + `img alt="Chorus App Mockup"` `:181` (rounded-2xl shadow) + grid `lg:grid-cols-4` `:184` cards: `AI Deep Dive` `Instant grammar analysis and CEFR-aligned drills…` `:190` `analytics` `:188`, `Real Talk` `AI-guided roleplays for real-world scenarios…` `:198` `forum` `:196`, `Teacher Marketplace` `Book 1:1 sessions with professional tutors…` `:206` `school` `:204`, `Phase 2 Ready` `High-fidelity voice & video calls with live translated captions…` `:216` `video_call` `:213` + badge `Coming Soon` `:211` (only this card) |
| 5 | **Pricing — 2 tiers** | `code.html:221-294` | `h2` `Simple, Transparent Pricing` `:225` + `p` `Start for free, upgrade when you're ready to accelerate.` `:226` + **Free** `:230` `$0/month` `:234` `Essential features to start your journey.` `280-character messages` `:242` `Basic AI translations` `:246` `Limited daily AI insights` `:250` CTA `Get Started Free` `:253` + **Premium** `:258` `$7.99/month` `:265` `Unleash the full power of the AI tutor.` `Most Popular` `:260` `1000-character messages` `:273` `Unlimited AI Deep Dives` `:277` `Monthly trial credits for live tutors` `:281` `Reduced marketplace fees` `:285` CTA `Upgrade to Premium` `:288` |
| 6 | **Mission** | `code.html:295-303` | `h2` `Our Mission` `:298` + `p` `We believe language shouldn't be a barrier, but a bridge. Chorus was built by a team of linguists and engineers…` `:299` |
| 7 | **Final CTA** | `code.html:304-315` | `bg-primary` `:305` dot pattern `:307` `h2` `Ready to reach fluency?` `:309` + `p` `Join thousands of learners… transformed their daily chats into a masterclass.` `:310` + CTA `Get Started Now` `:311` (`bg-surface-container-lowest` on `bg-primary`) |
| 8 | **Footer** | `code.html:317-346` | `Chorus` `:322` `© 2024 Chorus AI…` `:324` + `Product` `:330` (`Features` `:331` / `Pricing` `:332`) + `Company` `:335` (`About Us` `:336` / `Privacy Policy` `:337` / `Terms of Service` `:338`) + `Support` `:341` (`Help Center` `:342`) — 7 links, 3 columns |
| 9 | **No other sections** | — | v1 sections `How Chorus Works` (`#how-it-works`), `Supported Languages` (`#languages` TOP10 grid), `Enterprise` tier, language-selector hero badge, stats bar must be **deleted**. |

**Scope lock:** Only `frontend/src/pages/Landing.tsx:47` + `mobile/src/screens/LandingScreen.tsx:96` (+ shared nav/footer components if split out) are in scope for Home v2. No backend migration — Home remains static plus `GET /health` liveness (contract §6).

---

## 1. Slice Map — S-HOME-01..04 vs Wireframe Lines vs Master FR vs Backend vs Mobile Parity

### S-HOME-01 — Hero v2 (brain + headline + dual CTAs)

| Axis | Value |
|---|---|
| **Wireframe refs** | `wireframes/chorus_home_desktop_v2/code.html:134-163` hero section; `code.html:138` h1 `Communication is Learning.`; `code.html:139` span `Redefining how we acquire language.`; `code.html:141-143` bridging p; `code.html:145` `Start Your Journey`; `code.html:148-151` `Watch Demo` + `play_circle`; `code.html:154-158` brain hero `img alt="Brain Neural Pathways"` with `blur-3xl`; responsive `md:grid-cols-2` `:136` |
| **Current code refs (stale)** | `frontend/src/pages/Landing.tsx:47` `STRINGS.en.hero.titleA = 'Break Language Barriers'` + `:52` `titleB 'Connect Globally'` + `:53` subtitle `Real-time messaging…` + `:54` `cta 'Get Started Free'` + `:55` `seeHow`; hero render `frontend/src/pages/Landing.tsx:851-912`; `mobile/src/screens/LandingScreen.tsx:96-155` same stale hero (`Break Language Barriers`, `Connect Globally`, stats `Languages/Real-time/Free`, `Sparky` visual) |
| **REQUIREMENTS_MASTER.md FR refs** | Global DoD `§0` — mobile-first/web parity (NFR-22), no stubs; Phase 0 Foundation `§1` — runnable monorepo baseline; Phase 4 §5.2 `13.1` Credits & Access (pricing context for CTAs). Narrative refs: `REQUIREMENTS.md:14` Goals `Communication is Learning = messaging + learning same function`, `REQUIREMENTS.md:77` Phase 2+ differentiator (hero positioning) |
| **Backend contract** | **Static only + `GET /health`**. `backend/cmd/server/main.go:467` `r.GET("/health", appHealth.Liveness())` (liveness always 200 while process up, `backend/internal/observability/health.go:108` body `{status:healthy, version, commit, uptime_s, checks}`); LB probes `GET /health` `:467` not `/health/ready` (`health.go:5-8` liveness vs `health.go:9-12` readiness gate). No new endpoint. `backend/internal/observability/health.go:42-51` `commit=="dev"` fallback verified by `curl /health \| jq .commit == git rev-parse HEAD` in sign-off. |
| **Mobile parity** | Must ship together: `frontend/src/pages/Landing.tsx:851` hero section and `mobile/src/screens/LandingScreen.tsx:96` hero. One PR both surfaces (`crew/roles.py:97`). Mobile uses `SafeAreaView` + `ScrollView` with `ScrollToSection` pattern but same copy/assets/CTAs. Badge in v2 is removed (v1 badge `Available in {count} languages` is deleted). |

**Gherkin — S-HOME-01:**

```gherkin
@S-HOME-01 @home @hero @wireframe-chorus_home_desktop_v2
Feature: Home Hero v2 — Communication is Learning

  Background:
    Given I am unauthenticated
    And the backend GET /health returns 200 with {status: "healthy", checks: {postgres, redis, translation}}

  Scenario: Web hero renders v2 headline, subhead, dual CTAs, brain visual
    When I open "/" on web (frontend/src/pages/Landing.tsx:851)
    Then I see heading "Communication is Learning." as h1
    And I see heading subline "Redefining how we acquire language." as primary-colored span
    And I see paragraph "Bridging the gap between messaging apps and learning platforms. We turn your daily conversations into a personalized learning journey, making communication and learning the exact same function."
    And I see image with alt "Brain Neural Pathways"
    And I see button "Start Your Journey" (primary, href/cta to /register or /login when authed, or /waitlist per product decision — must match wireframe label)
    And I see button "Watch Demo" with icon "play_circle"
    And I do not see stale copy "Break Language Barriers" nor "Connect Globally"

  Scenario: Mobile hero parity
    When I open LandingScreen (mobile/src/screens/LandingScreen.tsx:96) on iOS/Android
    Then the same 5 elements render with native StyleSheet (no DOM), same strings, same brain asset (or native Image with same URL)
    And "Start Your Journey" navigates to Register/Login, "Watch Demo" opens demo (scroll or modal — consistent with web)
    And no stale strings from Landing.tsx:47-61 remain (verify grep zero hits)

  Scenario: Stale hero is purged
    Given I run grep for "Break Language Barriers|Connect Globally|Real-time messaging with instant translation|Available in {count} languages"
    Then the count is 0 in frontend/src/pages/Landing.tsx and mobile/src/screens/LandingScreen.tsx
```

**QA testRefs (failing first, then green — `docs/TDD_RESCUE_SPEC.md:12` / `docs/CREWAI_GAP_CLOSURE_PLAN.md:136`):**

| Suite | File | Locator (must fail until rebuild) |
|---|---|---|
| `e2e` | `e2e/tests/00-home.spec.ts:heroV2` | `await expect(page.getByRole('heading', {name: /Communication is Learning/})).toBeVisible()` + `await expect(page.getByText('Redefining how we acquire language')).toBeVisible()` + `await expect(page.getByRole('img', {name: /Brain Neural/})).toBeVisible()` + `await expect(page.getByRole('button', {name: 'Start Your Journey'})).toBeVisible()` + `await expect(page.getByRole('button', {name: 'Watch Demo'})).toBeVisible()` — fails today because Landing renders `Break Language Barriers` |
| `vitest` | `frontend/src/__tests__/Landing.test.tsx` | `render(<Landing/>)` → same heading/button/image queries; snapshot of hero section vs `chorus_home_desktop_v2/code.html:138` |
| `jest` | `mobile/__tests__/LandingScreen.test.tsx` or `App.test.tsx:hero` | `render(<LandingScreen/>)` → `getByText(/Communication is Learning/)` + `getByText(/Start Your Journey/)` + `getByTestId('brain-hero-image')`; screenshot comparison (optional) |
| `backend` | none | Static only; health tested via existing `backend/internal/observability/health_test.go:22` liveness 200 |

---

### S-HOME-02 — Bridging Section + 4-Card Ecosystem + App Mockup

| Axis | Value |
|---|---|
| **Wireframe refs** | `code.html:164-173` Bridging `h2` + `p` centered `max-w-4xl`; `code.html:174-220` Ecosystem: `code.html:178` `A Complete Language Ecosystem`, `code.html:179` `Everything you need… true fluency`, `code.html:180-182` app mockup `img alt="Chorus App Mockup"` (max-w-4xl shadow border), `code.html:184` grid `md:grid-cols-2 lg:grid-cols-4`, `code.html:186-192` AI Deep Dive (`analytics` primary), `code.html:193-199` Real Talk (`forum` secondary), `code.html:201-208` Teacher Marketplace (`school` tertiary), `code.html:210-217` Phase 2 Ready (`video_call` + `Coming Soon` badge `:211` + `surface-variant` circle) |
| **Current code refs (stale)** | `frontend/src/pages/Landing.tsx:63-74` features `STRINGS.en.features.items` 6 cards (Instant Translation / Grammar Analysis / Vocabulary Builder / Group Chats / Smart Search / Privacy First) rendered `Landing.tsx:914-933`; `mobile/src/screens/LandingScreen.tsx:20-37` `FEATURES` same 6 stale cards + `FEATURE_ACCENTS`; no mockup, no Bridging section, no 4-card order, no Coming Soon |
| **REQUIREMENTS_MASTER.md FR refs** | `§3.3` Audio calls with smart captions `8.2` captions/bookmark (Phase 2 Ready card teases `8.x`); `§4.3` 11.3 Scenario/real-talk hub + group-study hooks (Real Talk card); `§2.3` 3.4 Highlight new words + 3.5 Quality pipeline (AI Deep Dive); `§5.1` Teacher marketplace 12.x (Teacher Marketplace card previews 5.1). Global parity NFR-22 still applies. |
| **Backend contract** | Static only + `GET /health`. No new endpoint; cards are descriptive (link to `#features` / marketplace later per S-T-* slices). |
| **Mobile parity** | Cards must appear in **exact order** on both surfaces with same icons/titles/descs; `Coming Soon` badge only on Phase 2 Ready card; app mockup image centered above grid on both. Mobile uses `View` grid with `flexWrap` vs web `grid`. Old 6-card set deleted. |

**Gherkin — S-HOME-02:**

```gherkin
@S-HOME-02 @home @ecosystem @wireframe-chorus_home_desktop_v2:164
Feature: Home Ecosystem — Bridging + 4 Cards + Mockup

  Scenario: Bridging section renders
    When I open "/" 
    Then I see heading "Bridging Messaging and Learning." 
    And I see heading continuation "The Best of Both Worlds."
    And I see paragraph containing "Why choose between a messenger like WhatsApp and a learning tool like Duolingo?"

  Scenario: Ecosystem header + mockup + 4 cards in order
    When I scroll to "#features" (web id="features" code.html:175; mobile onLayout registerSection('features'))
    Then I see heading "A Complete Language Ecosystem"
    And I see paragraph "Everything you need to go from basic phrases to true fluency."
    And I see image alt "Chorus App Mockup" centered above the cards (max-w-4xl, rounded-2xl, shadow-lg, border-outline-variant/30)
    And I see exactly 4 cards in order:
      | title                | icon        | description contains                         | badge       |
      | AI Deep Dive         | analytics   | Instant grammar analysis and CEFR-aligned drills| none       |
      | Real Talk            | forum       | AI-guided roleplays for real-world scenarios | none       |
      | Teacher Marketplace  | school      | Book 1:1 sessions with professional tutors   | none       |
      | Phase 2 Ready        | video_call  | High-fidelity voice & video calls with live translated captions | Coming Soon |
    And card 4 badge "Coming Soon" is positioned top-right (absolute, bg-primary, rounded-bl-lg)
    And stale 6 cards "Instant Translation / Grammar Analysis / Vocabulary Builder / Group Chats / Smart Search / Privacy First" do not exist
    And stale section "How Chorus Works" does not exist

  Scenario: Mobile parity — same 4 cards
    When I open LandingScreen and scroll to features section
    Then the same 4 cards render in same order with same badge rule
    And icon circles use primary/secondary/tertiary/surface-variant tints matching web
```

**QA testRefs:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/00-home.spec.ts:ecosystem` | `expect(page.getByRole('heading', {name: 'A Complete Language Ecosystem'})).toBeVisible()` + `getByAltText('Chorus App Mockup')` + 4 `getByRole('heading', {name: /AI Deep Dive\|Real Talk\|Teacher Marketplace\|Phase 2 Ready/})` in order + `expect(page.getByText('Coming Soon')).toBeVisible()` + count 4 cards (not 6) |
| `vitest` | `frontend/src/__tests__/Landing.test.tsx` | `getAllByTestId('ecosystem-card').length === 4` + order assert + badge only on last |
| `jest` | `mobile/__tests__/LandingScreen.test.tsx` | `getAllByTestId('ecosystem-card')` 4 + badge test |
| `backend` | none | — |

---

### S-HOME-03 — Mission + Final CTA + Footer

| Axis | Value |
|---|---|
| **Wireframe refs** | `code.html:295-303` Our Mission (`id="about"` `:296`): `h2` `Our Mission` `:298`, `p` `We believe language shouldn't be a barrier, but a bridge…` `:299`; `code.html:304-315` Final CTA `py-32 bg-primary` `:305` dot pattern `:307` `h2` `Ready to reach fluency?` `:309` `p` `Join thousands of learners… masterclass.` `:310` CTA `Get Started Now` `:311`; `code.html:317-346` Footer `grid md:grid-cols-4` `:319` brand `Chorus` `:322` `© 2024 Chorus AI…` `:324` + `Product` `:330` (Features/Pricing), `Company` `:335` (About Us/Privacy Policy/Terms of Service), `Support` `:341` (Help Center) — 7 links |
| **Current code refs (stale)** | `frontend/src/pages/Landing.tsx:106-117` footer TAGLINE `Break language barriers…` + productLinks `Features/Web App` + companyLinks `How It Works/Languages` + supportLinks `API Status` `http://localhost:8080/health`; `:106-107` CTA `Ready to Break Language Barriers?` + `Get Started Now`/`Join Discord`; `:105-106` pricing note etc.; `mobile/src/screens/LandingScreen.tsx:259-288` `ctaSection` `Ready to Break Language Barriers?` + `ctaSubtitle` + footer `footerTagline` + `footerLinks` Pricing/About/Log In + `© 2026 Chorus…` — **all to be replaced** |
| **REQUIREMENTS_MASTER.md FR refs** | Global DoD `§0` no stubs, adapt `docs/RELEASE_GATE.md`; Narrative `REQUIREMENTS.md:122` GDPR/retention policy footer links (Privacy/Terms placeholder anchors acceptable for Home v2 — link targets are `#` per `code.html:337-342`); `docs/WIREFRAME_TRACE.md:27` `chorus_home` PASS row remains but note "implemented as v2" |
| **Backend contract** | Static only + `GET /health`. Footer Privacy/Terms/Help Center are `href="#"` placeholders in wireframe — must remain placeholders (no backend). |
| **Mobile parity** | Mission, final CTA dot-pattern (or native equivalent), footer 3 columns collapse to stacked on mobile but same 7 links/brand/copy. |

**Gherkin — S-HOME-03:**

```gherkin
@S-HOME-03 @home @mission @cta @footer @wireframe-chorus_home_desktop_v2:295
Feature: Home Mission + Final CTA + Footer

  Scenario: Mission section
    When I scroll to "#about"
    Then I see heading "Our Mission"
    And I see paragraph containing "We believe language shouldn't be a barrier, but a bridge. Chorus was built by a team of linguists and engineers dedicated to bridging global communication gaps through science-based acquisition, not rote memorization."

  Scenario: Final CTA
    When I scroll to the final primary section (bg-primary)
    Then I see heading "Ready to reach fluency?"
    And I see paragraph "Join thousands of learners who have transformed their daily chats into a masterclass."
    And I see button "Get Started Now" with bg surface-container-lowest text primary (inverted on primary)
    And final CTA background has dot pattern opacity 10% (or equivalent native decoration)

  Scenario: Footer — 7 links 3 columns + brand
    When I reach the footer
    Then I see brand "Chorus" and text "© 2024 Chorus AI. Language learning reimagined."
    And footer column "Product" contains links "Features" href="#features" and "Pricing" href="#pricing"
    And footer column "Company" contains "About Us" href="#about", "Privacy Policy" href="#", "Terms of Service" href="#"
    And footer column "Support" contains "Help Center" href="#"
    And there are exactly 7 footer links total
    And stale footer links "Web App", "How It Works", "Languages", "API Status" (http://localhost:8080/health) do not exist in footer
    And stale CTA "Ready to Break Language Barriers?" does not exist

  Scenario: Mobile footer parity
    When I scroll to footer on LandingScreen
    Then brand + © line + 7 links render (stacked layout acceptable) with same labels/hrefs
```

**QA testRefs:**

| Suite | File | Locator |
|---|---|---|
| `e2e` | `e2e/tests/00-home.spec.ts:footer` | `expect(page.getByRole('heading', {name: 'Our Mission'})).toBeVisible()` + `getByRole('heading', {name: 'Ready to reach fluency?'})` + `getByRole('button', {name: 'Get Started Now'})` + footer links count 7 + `expect(page.getByText('© 2024 Chorus AI')).toBeVisible()` + stale `expect(page.getByRole('heading', {name: 'Ready to Break Language Barriers'})).toHaveCount(0)` |
| `vitest/jest` | same suites | same locators native |
| `backend` | none | — |

---

### S-HOME-04 — Pricing Alignment (280 vs 1000 characters) + Premium Monthly $7.99

| Axis | Value |
|---|---|
| **Wireframe refs** | `code.html:221-294` pricing: `code.html:225` `Simple, Transparent Pricing`, `code.html:226` `Start for free…`, `code.html:230-256` Free card `$0/month` `:234` + 3 bullets `280-character messages` `:242` `Basic AI translations` `Limited daily AI insights`; `code.html:258-291` Premium card `border-primary` `Most Popular` `:260` `$7.99/month` `:265` + 4 bullets `1000-character messages` `:273` `Unlimited AI Deep Dives` `Monthly trial credits…` `Reduced marketplace fees` + CTAs `Get Started Free` `:253` / `Upgrade to Premium` `:288` |
| **Current code refs (stale)** | `frontend/src/pages/Landing.tsx:46` `STRINGS.en.pricing.free.features = ['Unlimited chats…', 'Live translation up to 200 characters' …]` + `:94` `Premium $79.90/year promo 2 months free features ['Automatic grammar analysis','Faster AI responses','Messages longer than 200 characters','Higher daily quotas']` + `enterprise` tier `100-103`; pricing render `:978-1038` 3 columns; `enterprise` card + yearly `$79.90` is **stale**; `mobile/src/screens/LandingScreen.tsx:209-257` same 3-tier + `200 characters`; `packages/shared/src/config.ts:14-16` `YEARLY_PRICE $79.90` is docs/upsell truth but Home v2 hero pricing must show **$7.99/mo** (monthly), not yearly |
| **REQUIREMENTS_MASTER.md FR refs** | **`REQUIREMENTS_MASTER.md:145` `13.1 Credits & Access model: Free=280-char, Premium=1000-char; 1 trial credit/mo; 10% fee for PM users`** — canonical. Also `REQUIREMENTS_MASTER.md:42` open item `Premium copy 280 vs 28` resolved as `280`. Backend hard caps `backend/internal/services/entitlement.go:40-48` `TranslationCharLimitFree=280 TranslationWordLimitFree=280 TranslationCharLimitPremium=1000 TranslationWordLimitPremium=1000 MessageWordLimitMax=10000` + `WordCount` `:53`. `REQUIREMENTS.md:139` Plans `P1 $7.99/mo, $79.90/yr 2 months free` / `P4a 280 words / 1000 words`. The Home v2 slice **renders characters** (`280-character` / `1000-character`) per wireframe even though server also enforces words — both must match 280/1000. |
| **Backend contract** | Static only + health, but **pricing copy must match entitlement caps**. `entitlement.go:42-48` is the source of truth; Home must not regress to `200-char` stale. `GET /health` still liveness; readiness `health.go:9` includes `translation` chain check — pricing change does not affect health. `GET /health` at `backend/cmd/server/main.go:467` cited as `GET /health:433` contract (canonical LB check `deploy/lb/haproxy.cfg:34 httpchk GET /health`). |
| **Mobile parity** | Both surfaces must show exactly 2 cards (Free + Premium monthly), same 7 total features (3+4), same CTAs, same `/month` unit. Yearly `$79.90` and Enterprise card are deleted from Home (yearly remains on `/pricing` if that page keeps it, but Home v2 does not). |

**Gherkin — S-HOME-04:**

```gherkin
@S-HOME-04 @home @pricing @monetization @wireframe-chorus_home_desktop_v2:221 @entitlement
Feature: Home Pricing — Free 280-char / Premium 1000-char monthly $7.99

  Scenario: Web pricing — 2 tiers with correct caps
    When I scroll to "#pricing"
    Then I see heading "Simple, Transparent Pricing"
    And I see paragraph "Start for free, upgrade when you're ready to accelerate."
    And I see Free card:
      | price | unit   | bullets |
      | $0    | /month | 280-character messages, Basic AI translations, Limited daily AI insights |
      And CTA "Get Started Free"
    And I see Premium card:
      | badge | price | unit   | bullets |
      | Most Popular | $7.99 | /month | 1000-character messages, Unlimited AI Deep Dives, Monthly trial credits for live tutors, Reduced marketplace fees |
      And CTA "Upgrade to Premium"
    And I do not see price "$79.90" in the Home pricing section
    And I do not see "200 characters" nor "Unlimited chats & groups" nor Enterprise tier
    And there are exactly 2 pricing cards

  Scenario: Pricing copy matches server entitlement caps
    Given backend entitlement.go has TranslationCharLimitFree=280 and TranslationCharLimitPremium=1000
    When I compare Home pricing bullets "280-character messages" and "1000-character messages"
    Then they match the server char caps (and word caps TranslationWordLimitFree=280 / Premium=1000)

  Scenario: Mobile pricing parity
    When I open LandingScreen pricing section
    Then Free $0/mo shows 280-character messages
    And Premium $7.99/mo (Most Popular) shows 1000-character messages + same 4 bullets
    And exactly 2 cards render
```

**QA testRefs:**

| Suite | File | Locator / assertion |
|---|---|---|
| `e2e` | `e2e/tests/00-home.spec.ts:pricing` | `expect(page.locator('#pricing').getByText('280-character messages')).toBeVisible()` + `getByText('1000-character messages')` + `getByText('$7.99')` + `getByText('Most Popular')` + `expect(page.locator('#pricing').locator('text=$79.90')).toHaveCount(0)` + card count 2 + no `Enterprise` |
| `vitest` | `frontend/src/__tests__/Landing.test.tsx` + `utils/__tests__/words.test.ts:words.test.ts:83` (word cap 280/1000 entitlement messaging) | `words.test.ts` already tests `entitlement.go:40-48` caps; pricing component test renders 2 cards with correct bullets |
| `jest` | `mobile/__tests__/LandingScreen.test.tsx` | same pricing bullets + price |
| `backend` | `backend/internal/services/entitlement_test.go` (unit) | `TranslationWordLimitFree==280` `TranslationWordLimitPremium==1000` `WordCount` + entitlement `Resolve` free/premium/self-host; already exists per `docs/TEST_PLAN.md:154` |
| `backend health` | `backend/internal/observability/health_test.go:22` + `health_test.go:51` | `/health` 200, `/health/ready` 503→200 after checks — unchanged |

---

## 2. Cross-Cutting Contracts (applies to all S-HOME-*)

### Backend contract — `GET /health` (`:433` symbolic = `main.go:467`)

```
GET /health          liveness  — always 200 while process up (LB healthcheck, deploy/lb/haproxy.cfg:34 httpchk GET /health)
GET /health/ready    readiness — 200 only when postgres+redis+translation chain ok, else 503 (Do not use for LB)

Response GET /health 200:
{
  "status": "healthy",
  "version": "2.0.0" (observability.Version),
  "commit": "<git SHA>" (buildCommit(), health.go:42, verified vs git rev-parse HEAD),
  "uptime_s": 123,
  "checkTime": "2026-09-03T00:00:00Z",
  "checks": {"postgres":"ok","redis":"ok","translation":"ok"}
}

Impl: backend/internal/observability/health.go:5 health surface (Liveness/Readiness), backend/cmd/server/main.go:455 NewHealth + :456 postgres check + :459 redis check + :464 translation check + :467 r.GET("/health", appHealth.Liveness()) + :469 /health/ready.
Health is also exposed at /api/v1/health in some docs but canonical LB/dev probes hit /health (Caddyfile deploy/dev/Caddyfile:26 @health path /health /health/ready, docker-compose.dev.yml:84 wget ... /health).
```

Home v2 itself requires **no new backend endpoint** — it is static. The only runtime backend assertion is that the page loads while `GET /health` is healthy (seed for e2e `e2e/acceptance/tests/p0-foundation.ts:17` `GET /health` 200, `e2e/global-setup.ts:19` `BACKEND_HEALTH`, `deploy/ci/artillery.perf.yml:35` `/health` perf probe).

### Mobile parity contract (`crew/roles.py:97` NFR-22)

- Every Home v2 element (TopNav, Hero, Bridging, Ecosystem 4 cards + mockup, Pricing 2 tiers, Mission, Final CTA, Footer) must exist on **both** `frontend/src/pages/Landing.tsx:851` and `mobile/src/screens/LandingScreen.tsx:96` in the same PR. `docs/WIREFRAME_TRACE.md:27` `chorus_home` + `28` `chorus_home_desktop` + `29` `chorus_home_desktop_v2` + `31-34` mobile variants all map to the same route (`/` web unauth, auth stack mobile) — they stay `PASS` but note "implemented as v2".
- Navigation: web `Fixed TopNav` (`position:sticky`) with anchors `href="#features" "#pricing" "#about"` matching `section id="features" id="pricing" id="about"`; mobile `TouchableOpacity onPress scrollToSection('features'|'pricing'|'about'|…)` (existing `sectionY`/`registerSection` pattern `LandingScreen.tsx:48-55`). Both must scroll correctly.
- i18n: Current `Landing.tsx:47` ships 10-language `STRINGS` (en/zh/hi/es/ar/fr/bn/pt/ru/ur) with full translation of stale hero/features/pricing. **Home v2 English is canonical** for BA sign-off; i18n re-translation of new hero/ecosystem/mission copy is **out of scope for S-HOME-01..04** (file follow-up slices if needed). Impl must either keep i18n plumbing but replace every string that overlaps v2 copy, or ship en-only for v2 sections with comment `// TODO S-HOME-i18n`. Stale translations of deleted sections must be deleted too — no orphan keys.
- Device boot: `.\start-android.ps1` boots `emulator-5554 device` + `adb shell getprop sys.boot_completed 1` + `curl /health` `commit == HEAD` (`backend/internal/observability/health.go:39` `CHORUS_BUILD_COMMIT`) — per `docs/TDD_RESCUE_SPEC.md:52` `S-SMOKE-02` and `docs/CREWAI_GAP_CLOSURE_PLAN.md:162`.

### File refs index

| Artifact | File ref |
|---|---|
| Canonical wireframe | `wireframes/chorus_home_desktop_v2/code.html:1` (full), hero `134-163`, bridging `164-173`, ecosystem `174-220`, pricing `221-294`, mission `295-303`, final CTA `304-315`, footer `317-346` |
| Stale web landing | `frontend/src/pages/Landing.tsx:1` (all), `STRINGS` `46-766` (10 langs), hero render `851-912`, features `914-933`, how `935-954`, languages `957-976`, pricing `978-1038`, cta `1040-1058`, footer `1060-1109`, `TOP10 44` |
| Stale mobile landing | `mobile/src/screens/LandingScreen.tsx:1` (all), `FEATURES 20-27` + `FEATURE_ACCENTS 29-36` + `STEPS 38-42`, hero `95-155`, features `157-175`, how `177-192`, languages `194-207`, pricing `209-257`, cta `259-268`, footer `270-288` |
| Master backlog | `REQUIREMENTS_MASTER.md:19` Global DoD, `§5.2` `13.1` Credits & Access 280/1000-char, `§1` Foundation P0 |
| Entitlement truth | `backend/internal/services/entitlement.go:40-48` char/word caps, `packages/shared/src/config.ts:14-16` display prices, `backend/internal/observability/health.go:5-12` liveness/readiness |
| Trace | `docs/WIREFRAME_TRACE.md:27-34` home rows, `docs/CREWAI_GAP_CLOSURE_PLAN.md:58-69` Cluster H slices |
| Health route | `backend/cmd/server/main.go:467` `/health` liveness, `:469` `/health/ready`, `deploy/lb/haproxy.cfg:34` `httpchk GET /health` |

---

## 3. Verification Checklist (attach to PR — `docs/CREWAI_GAP_CLOSURE_PLAN.md:158`)

### BA sign-off criteria — EXPLICIT (slice NOT DONE until all checked, `crew/phase_status.json:58` stays PENDING otherwise)

- [ ] **Wireframe visual match:** Device screenshot (Android AVD or iOS sim **and** web `npm run dev` snapshot) side-by-side with `chorus_home_desktop_v2` PNGs matches hero (brain + `Communication is Learning.` + `Redefining…` + both CTAs), Bridging centered text, Ecosystem mockup + 4 cards in order with only `Phase 2 Ready` badged `Coming Soon`, Pricing `Free $0 280-char` + `Premium $7.99 1000-char Most Popular`, Mission copy, Final CTA `Ready to reach fluency?` + `Get Started Now`, Footer 3 cols 7 links. BA initials + date.
- [ ] **Stale copy zero hits:** `grep -R "Break Language Barriers|Connect Globally|Real-time messaging with instant translation|Available in.*languages|Instant Translation.*Automatically|Grammar Analysis.*CEFR.*group|Vocabulary Builder.*Smart spaced|Group Chats.*100 participants|How Chorus Works|Supported Languages.*10 major|Ready to Break Language Barriers|Unlimited chats & groups|Live translation up to 200 characters|Automatic grammar analysis|Messages longer than 200 characters|Enterprise.*Self-hosted" frontend/src/pages/Landing.tsx mobile/src/screens/LandingScreen.tsx` returns 0; `grep -R "Enterprise" frontend/src/pages/Landing.tsx:978 mobile/src/screens/LandingScreen.tsx:209` 0 for Home pricing; `grep -R "79\.90" frontend/src/pages/Landing.tsx:978` 0 **inside Home pricing section** (yearly may survive only on `/pricing` page if that page keeps yearly plan — document exception in PR).
- [ ] **Pricing caps match entitlement:** `backend/internal/services/entitlement.go:42-48` char caps 280/1000 visible in Home bullets; `npm run -w @chorus/shared build` + `frontend vitest words.test.ts` green; no `200 characters` remains in Home.
- [ ] **Both surfaces built green:**
  - `cd backend && go vet ./...` exit 0
  - `cd backend && go test ./...` exit 0 (incl. `health_test.go:22`, `entitlement_test.go`)
  - `cd frontend && npm run build` (tsc && vite build) exit 0 + `grep -R "alice.dev" frontend/dist` 0 (NO_LEAK)
  - `cd frontend && npm test` (vitest) exit 0 (new `Landing.test.tsx` hero/ecosystem/pricing/footer queries green)
  - `cd mobile && npx tsc --noEmit` exit 0
  - `cd mobile && npm test` (jest) exit 0 (new `LandingScreen` queries green)
- [ ] **Automation green (TDD red→green proven):**
  - `e2e/tests/00-home.spec.ts` (new) — 4 specs (heroV2 / ecosystem / footer+mission+cta / pricing 280/1000) pass on dev stack (`e2e/global-setup.ts:19` `BACKEND_HEALTH=http://localhost:8080/health`)
  - Existing `frontend vitest` + `mobile jest` show exactly **+N** new pass, 0 fail
  - `e2e/acceptance/tests/p0-foundation.ts:17` `GET /health 200` still pass
- [ ] **Device-parity gate (not just `npm test`):**
  - `.\start-android.ps1` boots AVD `emulator-5554 device` + `adb shell getprop sys.boot_completed` == `1`
  - `curl -fsS http://localhost:8080/health | jq .commit` == `git rev-parse HEAD` (proves `backend/internal/observability/health.go:39` `CHORUS_BUILD_COMMIT` is fresh — `docs/TDD_RESCUE_SPEC.md:52` `S-SMOKE-02`)
  - On AVD, navigate Landing unauthenticated: hero visible, scroll to `#features`/`#pricing`/`#about` each reachable (web anchor + mobile `scrollToSection`), final CTA button taps to Register/Login (or Waitlist) without crash
  - `verify-wireframe-parity.sh` (or `docs/WIREFRAME_TRACE.md:58` audit) row for `chorus_home*` flipped from stale copy to `PASS (v2)` + GAP count note 62→? for home only
- [ ] **Mobile parity asserted:** Each locator in S-HOME-01..04 has both a `page.getBy*` (Playwright) and a `getByText`/`getByTestId` (jest/RNTL) — none is web-only. PR description lists `frontend/src/pages/Landing.tsx:851` and `mobile/src/screens/LandingScreen.tsx:96` lines touched.
- [ ] **`docs/WIREFRAME_TRACE.md:27-34` updated:** `chorus_home` `chorus_home_desktop` `chorus_home_desktop_v2` rows stay PASS with note "implemented as v2 — Communication is Learning + 4-card ecosystem + $7.99/mo pricing" + date + BA sign.
- [ ] **No secrets/outline leaks:** `.env*` untouched, no `WAITLIST_ADMIN_EMAILS` or `SMTP_PASSWORD` printed, `grep -R "sk-" frontend/dist mobile/dist` 0.
- [ ] **Gap sign-off Sheet:** BA signs `docs/GAP_SIGNOFF.md` (add row `S-HOME-01..04 | chorus_home_desktop_v2/code.html:134 | Landing.tsx:851 LandingScreen.tsx:96 | BA initials | date | commit SHA`) — only then `crew/state.py:97` `phase_complete()` may flip slice DONE (mirrors `docs/CREWAI_GAP_CLOSURE_PLAN.md:50` BA gate).

**Rejection rule:** If any box unchecked (e.g., device does not boot, commit mismatch, stale copy hit >0, e2e not run on real dev stack, mobile not touched), BA marks `CHANGES-REQUIRED` and slice returns to QA (new failing test added) per loop `docs/CREWAI_GAP_CLOSURE_PLAN.md:39` outer cycle.

---

## 4. Sequencing & Dependencies

```
S-HOME-01 (hero) → S-HOME-02 (ecosystem) → S-HOME-04 (pricing) → S-HOME-03 (mission/CTA/footer)
```
- S-HOME-01 is first — it proves rebuild vs freeze and deletes stale `STRINGS.en.hero` (`Landing.tsx:49-62`) so later slices cannot pass by mixing v1+v2.
- S-HOME-02 depends on 01 (shares `#features` anchor/id, mockup asset in same file — do in same PR ideally).
- S-HOME-04 can run parallel to 02 but must land before 03 (pricing is above mission in scroll order, `code.html:221` before `:295`; easier to diff as one Home rebuild PR).
- Recommended: **one PR for all 4 slices** (`Landing.tsx:47` full replace + `LandingScreen.tsx:96` full replace + `00-home.spec.ts` + `Landing.test.tsx`) — `docs/CREWAI_GAP_CLOSURE_PLAN.md:147` timeline `W1 BA spec → W1-2 QA failing tests → W2 impl green → verify_home_device PASS`. Splitting into 4 PRs is accepted only if each keeps device green (no intermediate GAP).

No backend ordering — `GET /health` already exists (`main.go:467`). No dependency on S-T-* (marketplace) though Teacher Marketplace card in S-HOME-02 is a tease whose links may target `#` until marketplace S-T-* ships.

---

## 5. Out of Scope (explicitly NOT in S-HOME-01..04)

- `chorus_premium_upgrade` page (`Pricing.tsx:1`) detailed checkout at `App.tsx:118` — pricing **Home** card copy only; `/pricing` page yearly `$79.90` logic may keep yearly but is not verified here (follow-up premium slice if yearly is retained).
- i18n re-translation of hero/ecosystem/mission into zh/hi/es/ar/fr/bn/pt/ru/ur — en only for BA sign-off; log `TODO S-HOME-i18n`.
- `Watch Demo` modal/video implementation — button must exist and be visible/tappable, but video URL/overlay is P2 nice-to-have (wireframe shows button `play_circle`, no target).
- `Teacher Marketplace` deep links from the card — `href="#features"` or `"/tutors"` placeholder acceptable until `S-T-01` ships.
- `Privacy Policy / Terms / Help Center` docs — footer anchors are `href="#"` per wireframe (content is separate slice).
- New backend work — none.

---

## 6. Traceability Summary

| Slice | Wireframe `code.html:line` | Current stale ref to purge | Master FR | Backend line | Mobile parity file |
|---|---|---|---|---|---|
| S-HOME-01 | `:138-158` hero + `:115-132` topnav | `Landing.tsx:49-55` title/badge + `LandingScreen.tsx:98-121` hero | DoD NFR-22 + §1 Foundation | `main.go:467` `/health` `health.go:108` liveness | `LandingScreen.tsx:96-155` |
| S-HOME-02 | `:164-173` bridging + `:174-220` ecosystem | `Landing.tsx:63-74` 6-card `features.items` + `:914-933` + `LandingScreen.tsx:20-37` `FEATURES` | §3.3 `8.2` captions, §4.3 `11.3` Real Talk, §5.1 `12.x` marketplace | `main.go:467` | `LandingScreen.tsx:157-175` |
| S-HOME-03 | `:295-315` mission+cta + `:317-346` footer | `Landing.tsx:106-117` cta+footer + `:1039-1109` + `LandingScreen.tsx:259-288` cta+footer | Global DoD + GDPR footer | `main.go:467` | `LandingScreen.tsx:259-288` |
| S-HOME-04 | `:221-294` pricing | `Landing.tsx:88-103` `pricing` `200 chars/$79.90/enterprise` + `:978-1038` + `LandingScreen.tsx:209-257` | `REQUIREMENTS_MASTER.md:145` `13.1` 280/1000 + `entitlement.go:40-48` | `entitlement.go:40` caps + `main.go:467` health | `LandingScreen.tsx:209-257` |

*BA signature line: ___________________________  Date: __________  Commit: __________  AVD screenshot attached: [ ] web  [ ] mobile*

