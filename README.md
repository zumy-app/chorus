# Chorus

**Learn a language while you communicate.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **Live now → [chorus.talk](https://chorus.talk)**

Chorus is a real-time messenger that breaks language barriers with built-in AI translation, grammar insights, and personalized learning — so every conversation makes you more fluent.

## Features

- **Real-time Translation** — Messages auto-translate into the recipient's native language. Sender sees original text; recipient sees their language. Powered by a provider chain with Redis caching for sub-500ms delivery.

- **AI Grammar Analysis** — Every message is analyzed for grammar patterns, corrections, and CEFR-aligned feedback. "Sparky" provides gentle, real-time corrections that turn conversations into learning moments.

- **Fluency Journey** — Track your progress with a personalized dashboard: messages translated, words learned, grammar concepts mastered, streaks, and daily goals.

- **Vocabulary Builder** — Tap any unknown word in a translated message to save it. Spaced repetition schedules reviews. A learned-word bank optimizes translations by skipping words you already know.

- **AI Writing Assistant** — Draft messages in your target language and get feedback, corrections, and completions before sending. Practice with AI-generated scenario role-play (restaurant, date, travel).

- **Grammar Deep Dive** — Tap any sentence for a full breakdown: verb conjugations, preposition rules, sentence structure analysis, and quick practice exercises.

- **Direct & Group Chats** — One-on-one conversations and group chats (2-100 participants) with real-time typing indicators, presence status, and read receipts.

- **Mobile-First Design** — Built with Capacitor for Android/iOS. The web app follows the same mobile-first UX. Optimized for one-handed use.

## Tech Stack

| Layer | Technology |
|-------|------------|
| **Backend** | Go 1.23+ · Gin · PostgreSQL 15 · Redis 7 · WebSocket · JWT |
| **Frontend** | React 18 · TypeScript · Vite · Tailwind CSS · Zustand |
| **Mobile** | Capacitor (Android/iOS) wrapping the React app |
| **Translation** | Google Translate API · OpenRouter · DeepSeek · Redis cache |
| **AI/LLM** | Ollama (local) · Cloud LLM fallback · Arize Phoenix (eval) |
| **Infra** | Docker Compose · Nginx L4 Load Balancer · Redis Pub/Sub |

## Quick Start

### Prerequisites

- Docker Desktop (running)
- Go 1.23+
- Node.js 20+

### One-Command Dev Start

```bash
.\start-dev.ps1
```

This starts everything in 3 windows:

| Window | What runs | Hot Reload |
|--------|-----------|------------|
| Docker | PostgreSQL, Redis, Ollama | — |
| Backend | Go server on `:8081` via `air` | Auto-rebuild on save |
| Frontend | Vite dev server on `:3000` | Instant HMR |

Then open **http://localhost:3000**.

### Manual Setup

```bash
# Docker services
docker-compose -f docker-compose.dev.yml up -d postgres-dev redis-dev ollama-dev

# Backend (Terminal 2)
cd backend && air

# Frontend (Terminal 3)
cd frontend && npm run dev
```

### Docker Production Build

```bash
docker-compose up -d --build
```

## Project Structure

```
chorus/
├── backend/
│   └── internal/
│       ├── handlers/     # HTTP handlers
│       ├── services/     # Business logic
│       ├── models/       # Data models
│       └── database/     # Migrations & connections
├── frontend/
│   └── src/
│       ├── components/   # React components
│       ├── pages/        # Route pages
│       ├── services/     # API & WebSocket clients
│       └── store/        # Zustand state
├── e2e/                  # Playwright tests
├── wireframes/           # UI design specs & prototypes
└── docker-compose.yml
```

## Development

### Tests

```bash
# Backend
cd backend && go test ./... -vet=off -count=1

# Frontend
cd frontend && npm test

# E2E (Playwright)
cd e2e && npx playwright test
```

### Useful Commands

| Task | Command |
|------|---------|
| Rebuild frontend | `docker-compose up -d --build frontend` |
| Rebuild backend | `docker-compose up -d --build backend` |
| View logs | `docker-compose logs -f backend` |
| Stop all | `docker-compose down` |
| Reset database | `docker-compose down -v && docker-compose up -d` |

## Roadmap

- [ ] Frontend parity for all backend APIs (presence, search, grammar, vocabulary, calls)
- [ ] Mobile UX hardening for Android/iOS
- [ ] Observability & health diagnostics
- [ ] Horizontal scaling & infrastructure automation
- [ ] Premium plans (PayPal subscriptions)
- [ ] Live tutoring marketplace (Phase 2)
- [ ] Public language-pair group chats (Phase 2)

## Contributing

We're actively looking for contributors to help roll out Chorus. If you're a **software engineer, SRE, product manager, linguist, or educator** — we'd love your help.

```bash
git clone https://github.com/zumy-app/chorus.git
cd chorus
.\start-dev.ps1
```

Open an issue to get started or pick something from the roadmap above.

## License

[MIT](LICENSE)
