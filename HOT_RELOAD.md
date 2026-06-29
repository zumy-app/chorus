# 🔥 Hot Reload Guide — Frontend Development

Edit `.tsx`/`.ts` files and see changes **instantly** in the browser — no Docker rebuilds, no redeploys.

## How It Works

Vite's dev server uses **Hot Module Replacement (HMR)**. When you save a file, Vite sends only the changed module to the browser, which swaps it in without a full page refresh. State is preserved.

## Setup

### 1. Start Backend Services (Docker)

Keep PostgreSQL, Redis, and the Go backend running in Docker:

```powershell
cd C:\dev\chorus
docker-compose up -d
```

This starts: `postgres`, `redis`, `backend`, `frontend` (production nginx build — we'll ignore this).

### 2. Start Frontend Dev Server (Local)

In a **separate terminal**:

```powershell
cd C:\dev\chorus\frontend
npm run dev
```

You'll see:
```
VITE v5.x.x  ready in xxx ms
➜  Local:   http://localhost:5173/
➜  Network: http://192.168.x.x:5173/
```

### 3. Open the App

Open **http://localhost:5173** in your browser.

### 4. Edit Any File

Try changing something in `frontend/src/pages/Landing.tsx` or any `.tsx` file. Save it — the browser updates instantly.

## How the Proxy Works

The Vite dev server at `localhost:5173` proxies API and WebSocket requests to the Docker backend:

| Browser URL | Proxied To |
|-------------|-----------|
| `http://localhost:5173/api/*` | `http://localhost:8080/api/*` (Go backend) |
| `ws://localhost:5173/ws` | `ws://localhost:8080/ws` (WebSocket) |

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

## What About the Docker Frontend Container?

The Docker `frontend` container (port 3000) is still running with the production nginx build. You can ignore it while using the dev server. When you're done with development, just stop the `npm run dev` process.

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `npm run dev` fails with missing modules | Run `npm install` first |
| API calls return 404 | Ensure `docker-compose up -d` is running (backend on 8080) |
| WebSocket won't connect | Check that backend is healthy: `curl http://localhost:8080/health` |
| Port 5173 already in use | Kill the process or change port in `vite.config.ts` |
| Changes not showing | Make sure you're on http://localhost:5173, not http://localhost:3000 |

## Stopping

```powershell
# Stop the dev server
Ctrl+C in the terminal where npm run dev is running

# Stop Docker services (when done developing)
docker-compose down
```
