# Chorus E2E Tests

End-to-end test suite for the Chorus Multilingual Messenger using **Playwright**.

## 🎯 What These Tests Cover

| Suite | Tests | Description |
|-------|-------|-------------|
| `01-auth` | 5 | Login, logout, session persistence, invalid credentials |
| `02-chat-creation` | 4 | Direct chat creation, chat list, duplicate prevention |
| `03-messaging-translation` | 6 | ⭐ **Core**: Cross-language messaging, real-time delivery, translation verification |
| `04-grammar` | 7 | Grammar breakdown panel, patterns, word-by-word, difficulty badge |
| `05-ai-tutor` | 9 | AI Tutor panel, breakdown, examples, flashcards, custom Q&A |
| `06-vocabulary` | 5 | Save words, vocabulary list, stats, practice flow |
| `07-search` | 4 | Message search, results, empty state |
| `08-settings` | 7 | Profile settings, language selection, target languages |
| `09-realtime` | 4 | WebSocket connection, typing indicators, real-time delivery |
| `10-health` | 9 | Backend health, API endpoints, translator-engine, console errors |
| `11-grammar-ai-local` | 2 | ⭐ **Local-first**: grammar AI served by local Ollama `qwen2.5:3b` (`ollama_local`), structured analysis + caching |
| **Total** | **62** | |

## 📋 Prerequisites

1. **Docker Desktop** running
2. **Node.js 20+** installed
3. **Test users** must exist in the database:
   - `uhsarp@gmail.com` (English speaker, password: `Demor@cer1`)
   - `avcxafefwer@gmail.com` (Spanish speaker, password: `Demor@cer1`)

   If they don't exist yet, register them once via the UI at `http://localhost:3000/register`.

## 🚀 Quick Start

```bash
# 1. Install dependencies
cd e2e
npm install

# 2. Install Playwright browser
npx playwright install chromium

# 3. Run all tests (starts Docker services automatically)
npm test

# 4. View test report
npm run test:report
```

## 🖥️ Running Tests

### Run all tests (headless)
```bash
npm test
```

### Run tests with visible browser
```bash
npm run test:headed
```

### Run tests in Playwright UI mode (interactive)
```bash
npm run test:ui
```

### Run a specific test suite
```bash
npx playwright test 01-auth
npx playwright test 03-messaging-translation
npx playwright test 11-grammar-ai-local
```

### Run only the core suites (auth, chat, messaging, grammar, AI tutor)
```bash
npm run test:core
```

### Debug a specific test
```bash
npx playwright test 03-messaging-translation --debug
```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `E2E_BASE_URL` | `http://localhost:3000` | Frontend URL |
| `E2E_API_URL` | `http://localhost:8080/api/v1` | Backend API URL |
| `E2E_SKIP_STARTUP` | `false` | Skip `docker-compose up` (use if services already running via `start-dev.ps1`) |
| `E2E_STOP_SERVICES` | `false` | Run `docker-compose down` after tests |

### Examples

```bash
# If services are already running (e.g., via start-dev.ps1), skip startup
E2E_SKIP_STARTUP=true npm test

# Run against a different frontend port (e.g., Vite dev mode)
E2E_BASE_URL=http://localhost:5173 npm test

# Stop services after tests complete
E2E_STOP_SERVICES=true npm test
```

### Startup Behaviour

The `global-setup.ts` script is now more resilient:
1. **Checks if backend is already running** — if port :8080 responds with `healthy`, it skips all Docker startup entirely
2. **Ensures Docker Desktop** — if Docker CLI doesn't respond, it attempts to launch Docker Desktop automatically and waits up to 60s
3. **Tries dev compose first** — prefers `docker-compose.dev.yml` over `docker-compose.yml`, so it works with the development stack
4. **Graceful fallback** — if no compose file works, it prints a helpful message suggesting `E2E_SKIP_STARTUP=true` and continues waiting for already-running services

## 🏗️ Architecture

```
e2e/
├── playwright.config.ts          # Playwright configuration
├── global-setup.ts               # Starts Docker services, waits for health
├── global-teardown.ts            # Optional service cleanup
├── fixtures/
│   ├── users.ts                  # Test user credentials
│   └── test-helpers.ts           # Shared utilities (login, send message, etc.)
├── tests/
│   ├── 01-auth.spec.ts
│   ├── 02-chat-creation.spec.ts
│   ├── 03-messaging-translation.spec.ts  ⭐ Core
│   ├── 04-grammar.spec.ts
│   ├── 05-ai-tutor.spec.ts
│   ├── 06-vocabulary.spec.ts
│   ├── 07-search.spec.ts
│   ├── 08-settings.spec.ts
│   ├── 09-realtime.spec.ts
│   ├── 10-health.spec.ts
│   └── 11-grammar-ai-local.spec.ts  ⭐ Local qwen grammar analysis
├── package.json
└── tsconfig.json
```

## 🔑 Key Design Decisions

### Two-Browser-Context Pattern
The core messaging tests (Suite 3) use **two separate browser contexts** to simulate the English and Spanish users chatting simultaneously. This mirrors real-world usage where two different devices chat with each other.

```typescript
const senderContext = await browser.newContext()   // English user
const receiverContext = await browser.newContext()  // Spanish user
```

### Async Translation Waiting
Translations arrive asynchronously via WebSocket (`message_updated` event) after the initial `new_message`. Tests use a generous timeout (5 min) to account for ALMA-7B GGUF model download on first start:

```typescript
await waitForTranslation(receiverPage, testMsg, 90_000)
```

### Sequential Execution
Tests run sequentially (`workers: 1`) because they share state (users, chats). Parallel execution would cause conflicts.

## 🐛 Troubleshooting

### "Login failed" errors
- Verify test users exist: try logging in manually at `http://localhost:3000/login`
- Check backend is running: `curl http://localhost:8080/health`

### Translation tests timeout
- Synatra-7B GGUF model may be downloading on first run (can take 5+ minutes, ~4.14 GB)
- Check translator-engine container: `docker logs chorus-translator-engine`
- Check dev translator-engine container: `docker logs chorus-dev-translator-engine`
- Translation cache: unique messages (with `Date.now()`) avoid cache hits

### AI Tutor tests fail
- Ollama must be running: `docker logs chorus-ollama`
- Model must be pulled: `docker exec chorus-ollama ollama list`
- If Ollama is down, grammar falls back to regex (tests 4.x still pass, 5.x may fail)

### Grammar AI (11.x) tests are slow or timeout
- Local CPU inference of `qwen2.5:3b` takes ~1-2 min per analysis — that is expected.
- First run also warms the model; later calls are faster.
- The model must be the one the backend is configured with. Verify the backend
  picks up the local chain:
  - `backend/.env`: `GRAMMAR_ANALYSIS_PROVIDER_ORDER=ollama_local` and
    `PROVIDER_OLLAMA_LOCAL_MODEL=qwen2.5:3b` (must match the model pulled by the
    `ollama` docker service in `docker-compose.yml`).
  - If you changed `backend/.env`, restart the backend (air hot-reload watches
    `.go` files, not `.env`).
- If generation is < 5 tok/s, check the Ollama container CPU limit
  (`docker inspect chorus-ollama --format '{{.HostConfig.NanoCpus}}'`). The
  compose file must allow most cores (`deploy.resources.limits.cpus`).

### WebSocket tests fail
- Check backend logs: `docker logs chorus-backend`
- Verify WS endpoint: `ws://localhost:8080/ws`
- Browser console should show "WebSocket connected"

### Port conflicts
- Frontend: 3000, Backend: 8080, Translator-Engine: 5002, Ollama: 11434
- Stop conflicting services or change ports in `docker-compose.yml`

## 📊 Test Report

After running tests, view the HTML report:
```bash
npm run test:report
```

The report includes:
- Pass/fail status for each test
- Screenshots on failure
- Video recordings of failed tests
- Playwright traces for debugging