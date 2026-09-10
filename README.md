# Go-Ledger

A double-entry-ish ledger API in Go behind an IP rate limiter and a security
logger, with a React console and demo scenarios that attack it.

**[Try the console](https://mariaalexissales.github.io/Go-Ledger/)** — a recording, not a live server.

> **This project contains a deliberate vulnerability.** `CLIENT_IP_MODE`
> defaults to `xff-trust-all`, which trusts `X-Forwarded-For` verbatim and makes
> the rate limiter trivially bypassable. `/ops` has no authentication. See
> [SECURITY.md](SECURITY.md).

## Quickstart

```bash
docker compose up --build
```

<http://localhost:8080>. Postgres, migrations, seed data and the console on one port.

With hot reload, in two terminals:

```bash
npm run db:up
npm run dev:api
npm run dev:web
```

<http://localhost:5173>. Vite proxies `/api` and `/ops` to the Go server.

## Routes

```
/health          unguarded   container healthcheck
/api/*           GUARDED     the ledger: accounts, transactions
/ops/*           unguarded   the console watching the guard
/*               unguarded   the React app
```

## Demos

Open **Demos** in the console. Each card fires scripted traffic at `/api` from
the server itself, using synthetic source IPs, so your own browser is never
rate-limited by a run.

| Scenario       | What it does                                            | What it proves                                                         |
| -------------- | ------------------------------------------------------- | ---------------------------------------------------------------------- |
| `baseline`     | 4 clients, 3 requests each, at a human pace             | What a healthy log looks like.                                         |
| `burst`        | One IP, `limit × 2 + 5` requests back to back           | Refused after the limit, with a `Retry-After` countdown.               |
| `low-and-slow` | 20 clients, 6 requests each, none near the limit        | More traffic than `burst`, zero blocks. Per-IP limiting cannot see it. |
| `xff-spoof`    | One machine, a new forged `X-Forwarded-For` per request | Zero blocks. The guard believes the header.                            |
| `enumeration`  | Sequential `GET /api/accounts/{1..N}` from one IP       | Volume gets blocked; the shape is what a log is for.                   |

Flip **Trust X-Forwarded-For** off and re-run `xff-spoof`: the same 90 requests
go from 0 blocked to 60. Scenarios live in
[internal/demo/scenarios.go](internal/demo/scenarios.go).

## API

Lists return `{"data": [...], "total": N, "limit": N, "offset": N}`. Errors
return `{"error": "..."}`. `amount` and `balance` are JSON numbers, not strings.

### Ledger: `/api`, rate limited

| Method | Path                              | Notes                                 |
| ------ | --------------------------------- | ------------------------------------- |
| GET    | `/api/accounts`                   | `limit`, `offset`, `q` (name search)  |
| POST   | `/api/accounts`                   | `{"name": "..."}`                     |
| GET    | `/api/accounts/{id}`              |                                       |
| DELETE | `/api/accounts/{id}`              | Cascades to its transactions          |
| GET    | `/api/accounts/{id}/transactions` | `limit`, `offset`                     |
| GET    | `/api/transactions`               | `limit`, `offset`, `account_id`       |
| POST   | `/api/transactions`               | `{"account_id": 1, "amount": 100.50}` |
| GET    | `/api/transactions/{id}`          |                                       |
| DELETE | `/api/transactions/{id}`          |                                       |

### Console: `/ops`, not rate limited, no auth

| Method | Path                         | Notes                                                                                    |
| ------ | ---------------------------- | ---------------------------------------------------------------------------------------- |
| GET    | `/ops/config`                | Active IP mode, limiter policy, and how the guard sees _you_                             |
| GET    | `/ops/events`                | `flag_status`, `ip_address` (comma-separated), `action_type`, `since`, `limit`, `offset` |
| GET    | `/ops/events/stream`         | SSE, honors `Last-Event-ID`, heartbeats every 15s                                        |
| GET    | `/ops/stats`                 | `window`, totals, top IPs, per-minute buckets, currently-blocked IPs                     |
| GET    | `/ops/demos`                 | The five scenarios and their metadata                                                    |
| POST   | `/ops/demos/{id}/run`        | Returns the full step list and a verdict                                                 |
| PUT    | `/ops/config/client-ip-mode` | `{"mode": "xff-trust-all" \| "remote-addr"}`                                             |
| PUT    | `/ops/config/limiter-policy` | `{"limit": 30, "window": "10s", "block_period": "30s"}`                                  |
| POST   | `/ops/events/reset`          | Truncates `security_events`                                                              |

The three mutating routes exist only when `DEMOS_ENABLED=true`.

## Configuration

Every value has a working default except `DATABASE_URL`. The full list lives in
[.env.example](.env.example) — copy it to `.env`.

```bash
go run ./cmd/server [serve|seed|reset|healthcheck]
```

Other scripts: `seed`, `reset`, `build`, `fmt`, `format`, `typecheck`, `test`,
`test:race`, `lint`, `record`, `build:pages`, `preview:pages`, `up`, `down`,
`db:down`.

## GitHub Pages

Pages serves static files only, so the published console runs in replay mode:
`cmd/record` drives a real local server and writes each scenario's genuine
output to `web/public/replay/`.

One-time setup: **Settings → Pages → Source → GitHub Actions**. To refresh the
recordings: **Actions → Record demo data → Run workflow**.

## License

[MIT](LICENSE). The deliberate `X-Forwarded-For` vulnerability is documented in
[SECURITY.md](SECURITY.md).
