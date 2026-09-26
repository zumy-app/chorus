# Layer 4 Load Balancer — leastconn (10.1, 10.4)

Stateless chat servers behind a TCP LB. No sticky sessions; any replica serves any connection.

## Stateless checklist (10.4)

All durable state is in Postgres or Redis; replicas hold only ephemeral connection handles.

| State | Store | Notes |
|-------|-------|-------|
| Messages, chats, users, receipts, vocab | Postgres | `persist before ack` — message is acked only after DB insert |
| Translation cache, grammar cache | Redis `translation:*` | 24h TTL, shared |
| WebSocket connection registry | Redis `ws:registry:{userId}` → `{serverId,connId}` | TTL 45s, heartbeat 15s, Lua-guarded refresh/unregister |
| Presence | Redis `presence:{userId}` + `online_users` | 5m TTL, DB log is audit only |
| Rate-limit counters | Redis `ratelimit:*` | INCR+PEXPIRE Lua, fallback to in-memory when Redis unavailable |
| Pub/Sub fan-out | Redis `user:{id}`, `chat:{id}`, `server:{id}` | Cross-server delivery, typing, calls |
| Upload bytes | Shared `uploads_data` volume (`/root/uploads`) | Single-host compose shares named volume; multi-host needs NFS/S3 |
| JWT, refresh tokens | Postgres + client localStorage | Stateless verification via `JWT_SECRET` env |

No replica-local maps are consulted for routing — `DeliveryRouter` always looks up the registry first, then falls back to local `hub.SendToUser` only when the recipient is local or has no registry entry.

## Topology

```
client ──► LB :8080 (TCP, leastconn) ──► backend:8080 (N replicas)
              │ health: GET /health every 5s
              └─► stats :8404 /stats
```

Frontend `nginx.prod.conf` proxies `/api`, `/ws`, `/media` to `lb:8080`, not directly to `backend`.

## Algorithm

`leastconn` — WebSockets are long-lived, so balancing by open connections approximates spare capacity better than round-robin.

Env vars (compose):

- `LB_ALGORITHM=leastconn|least-load` (default `leastconn`)
- `LB_MAXCONN_PER_SERVER=1000`
- `LB_HEALTHCHECK_INTERVAL=5s`

Current: HAProxy `balance leastconn` + `maxconn 1000` per server + `httpchk GET /health` (rise 2 fall 3).

## Horizontal scale

Backend is stateless (state in Postgres/Redis). Scale with compose:

```bash
docker compose -f docker-compose.prod.yml up -d --scale backend=3
# or swarm
docker stack deploy -c docker-compose.prod.yml chorus
```

LB uses `server-template 6 backend:8080 resolvers docker` so new IPs are discovered via the embedded DNS `127.0.0.11` without config reload. `nginx.conf` (stream `least_conn`) is an equivalent Nginx alternative.

## Health

Backend exposes `/health` (liveness 200) and `/health/ready` (503 until DB/Redis/translation ready). LB uses `/health` so a degraded dependency does not kill the instance prematurely; `ready` gates rolling deploys.

## No stickiness

Verified: no `cookie`, `stick-table`, or `ip_hash`. Redis `ws:registry` + `server:{id}` Pub/Sub routes cross-server delivery instead.

## Verify

```bash
docker compose -f docker-compose.prod.yml config --quiet
curl -f http://localhost:8404/stats  # HAProxy stats
curl -f http://lb:8080/health
```
