# 🔥 Hot Reload Guide — Development

Edit `.tsx`/`.ts` files and see changes **instantly** in the browser, and `.go` changes auto-rebuild the backend — no Docker rebuilds, no redeploys.

## How It Works

- **Frontend**: Vite's dev server uses **Hot Module Replacement (HMR)**. When you save a file, Vite sends only the changed module to the browser, which swaps it in without a full page refresh. State is preserved.
- **Backend**: [`air`](https://github.com/air-verse/air) watches `.go` files and recompiles/restarts the server on every save (~1s rebuild).

## One-Command Start

From the repo root, run:

```powershell
.\start-dev.ps1
```

This opens 3 windows:
- **Docker** (background): PostgreSQL, Redis, Ollama
- **Backend** (new terminal): Go server with `air` on port **8080**
- **Frontend** (new terminal): Vite dev server on port **3000**

## Manual Setup

### 1. Start Backend Services (Docker)

```powershell
cd C:\dev\chorus
docker-compose -f docker-compose.dev.yml up -d postgres-dev redis-dev ollama-dev
```

This starts the development Docker services (all on different ports to avoid conflicts with production).

### 2. Start Go Backend (with hot-reload)

```powershell
cd C:\dev\chorus\backend
air
```

Every time you save a `.go` file, `air` recompiles and restarts the server automatically.

### 3. Start Frontend Dev Server (with HMR)

In a **separate terminal**:

```powershell
cd C:\dev\chorus\frontend
npm run dev
```

You'll see:
```
VITE v5.x.x  ready in xxx ms
➜  Local:   http://localhost:3000/
```

### 4. Open the App

Open **http://localhost:3000** in your browser.

### 5. Edit Any File

- **Frontend** (`.tsx`/`.ts`/`.css`): Save — the browser updates instantly.
- **Backend** (`.go`): Save — `air` rebuilds and restarts automatically.

## How the Proxy Works

The Vite dev server at `localhost:3000` proxies API and WebSocket requests to the backend (running on port 8081 in dev mode):

| Browser URL | Proxied To |
|-------------|-----------|
| `http://localhost:3000/api/*` | `http://localhost:8081/api/*` (Go backend) |
| `ws://localhost:3000/ws` | `ws://localhost:8081/ws` (WebSocket) |

This is configured in `frontend/vite.config.ts`:
```typescript
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true,
  },
  '/ws': {
    target: 'ws://localhost:8080',
    ws: true,
  },
},
```

> Note: The vite.config.ts targets port 8080 (production). When running in dev mode (port 8081), set `$env:VITE_API_URL=http://localhost:8081` or update the proxy target accordingly.

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `npm run dev` fails with missing modules | Run `npm install` first |
| API calls return 404 | Check backend is running: `curl http://localhost:8081/health` |
| WebSocket won't connect | Check backend logs for WebSocket setup |
| Port 3000 already in use | Kill the process or change `server.port` in `vite.config.ts` |
| `air` not found | Run `go install github.com/air-verse/air@latest` |

## Stopping

```powershell
# Close the backend and frontend terminal windows (or Ctrl+C in each)
# Stop Docker services
docker-compose -f docker-compose.dev.yml down
```
