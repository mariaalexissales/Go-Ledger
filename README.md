# Go-Ledger

**[Try the console](https://mariaalexissales.github.io/Go-Ledger/)** _(a recording, see
[GitHub Pages](#github-page-preview))_

A small double-entry-ish ledger API in Go, wrapped in an IP rate limiter and a
security logger. It comes with a React console that makes the logger visible and a
set of demo scenarios that attack it on demand.

The ledger is the excuse. The interesting part is `internal/ops`. Every request is
evaluated by a rate limiter and recorded as `ALLOWED` or `BLOCKED`, and the console
lets you watch that happen live while scripted traffic patterns try to get past it.

```mermaid
flowchart TD
    browser(["Browser"])
    runner["Demo runner<br/>POST /ops/demos/{id}/run"]
    router{{"chi router · one port"}}
    ui(["React console"])

    browser --> router
    runner -. "loopback · synthetic source IPs" .-> router

    router -- "/api/* · GUARDED" --> resolver
    router -- "/ops/* · unguarded" --> ops
    router -- "/* · SPA" --> ui

    subgraph guard ["SecurityLogger"]
        direction LR
        resolver["Resolver.ClientIP<br/>demo token → XFF → RemoteAddr"] --> limiter{"RateLimiter<br/>.Allow"}
        limiter -- ALLOWED --> ledger["Ledger handlers"]
        limiter -- BLOCKED --> deny["429 · Retry-After"]
    end

    limiter == "recorded either way" ==> recorder["Recorder · batches of 200<br/>INSERT … RETURNING id"]
    recorder --> db[("security_events")]
    db --> stream["Hub → /ops/events/stream<br/>SSE · Last-Event-ID replay"]

    ops["Console API<br/>events · stats · config · demos"] --> ui
    stream --> ui
```

The guard is `/api` only, and the console watching it is not behind the guard. Both
halves are detailed in [How the security logger works](#how-the-security-logger-works).

> **This project contains a deliberate vulnerability.** The default client-IP
> resolver trusts the `X-Forwarded-For` header verbatim, which makes the rate
> limiter trivially bypassable. That is the point of the `xff-spoof` demo. You can
> toggle it off in the UI and re-run to see the difference. Do not copy
> `internal/ops/clientip.go` into anything real without reading the note in it.

---

## Quickstart

### Everything in one command

```bash
docker compose up --build
```

Then open <http://localhost:8080>. Postgres comes up first, gated on a healthcheck.
Migrations run, 50 accounts and 200 transactions are seeded, and the Go binary serves
both the API and the console on a single port, so no CORS is involved at all.

### Development with hot reload

```bash
npm run db:up          # Postgres only
npm run dev:api        # terminal 1 — Go API on :8080
npm run dev:web        # terminal 2 — Vite on :5173
```

Open <http://localhost:5173>. Vite proxies `/api` and `/ops` to the Go server, so the
app behaves identically to the container build while keeping HMR. Two terminals because
running both from one npm script needs a dependency, and PowerShell cannot portably
background a process — the root `package.json` has no dependencies and needs no
`npm install`.

Copy `.env.example` to `.env` if you want to change anything. A missing `.env` is not
fatal, and real environment variables always win.

Other scripts: `seed`, `reset`, `build`, `fmt`, `format`, `typecheck`, `test`,
`test:race`, `record`, `build:pages`, `preview:pages`, `up`, `down`, `db:down`.
`npm run lint` wants [golangci-lint](https://golangci-lint.run)
(`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`); CI runs
the same config plus `go test ./... -race`, which needs cgo and so will not run locally
without a C compiler.

---

## How the security logger works

Every request to `/api` passes through `SecurityLogger`
([internal/ops/middleware.go](internal/ops/middleware.go)):

1. **Resolve the source IP** using the configured `CLIENT_IP_MODE`.
2. **Ask the rate limiter** whether that IP is over its budget.
3. **Record the decision** as a `security_events` row, always, allowed or blocked.
4. **Refuse with 429** (JSON, plus `Retry-After` and `X-RateLimit-*`) if blocked.

### The router is split in two, on purpose

```
/health          unguarded   container healthcheck
/api/*           GUARDED     the ledger: accounts, transactions
/ops/*           unguarded   the console watching the guard
/*               unguarded   the React app (embedded SPA, history fallback)
```

The guard protects the ledger, and the console is the instrument observing it. If the
console were guarded too, the dashboard would rate-limit itself within seconds and
flood the event log with its own reads, burying the traffic it exists to display.
This also means the only rows in `security_events` are real ledger traffic, which is
what makes a demo run legible.

The tradeoff is that **`/ops` has no authentication.** It is fine for a local demo and
unacceptable anywhere else. Set `OPS_ENABLED=false` and `DEMOS_ENABLED=false` before
deploying this anywhere reachable.

### Events reach the browser without polling

Audit rows go through a batching recorder
([internal/ops/recorder.go](internal/ops/recorder.go)): one worker, a bounded queue,
inserts of up to 200 rows at a time. Subscribers are published to only _after_ the
insert, so every streamed event carries a real database id — which is what lets the
browser reconnect with `Last-Event-ID` and have the gap replayed instead of lost. A
goroutine per request would exhaust the pool during a burst and drop the very events
the demo exists to show.

---

## Demo scenarios

Open **Demos** in the console. Each card fires a scripted traffic pattern at the API
from the server itself, using synthetic source IPs from reserved ranges -- the RFC
5737 documentation blocks, plus RFC 2544 benchmarking space for the forged headers
in `xff-spoof` -- so your own browser is never rate-limited by a run.

| Scenario       | What it does                                            | What it proves                                                                                    |
| -------------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `baseline`     | 4 clients, 3 requests each, at a human pace             | What a healthy log looks like. Read the rest against it.                                          |
| `burst`        | One IP, `limit × 2 + 5` requests back to back           | The case the limiter is built for: refused after the limit, with a `Retry-After` countdown.       |
| `low-and-slow` | 20 clients, 6 requests each — 120 in total, none near the limit | **More total traffic than `burst`, and zero blocks.** Per-IP limiting cannot see distributed load. |
| `xff-spoof`    | One machine, a new forged `X-Forwarded-For` per request | **Zero blocks.** The guard believes the header, so the attacker gets a fresh identity every time.  |
| `enumeration`  | Sequential `GET /api/accounts/{1..N}` from one IP       | Volume gets blocked, but the shape (ordered IDs, a trail of 404s) is what a log is for.            |

The two that show the guard failing are the more useful ones.

### The before and after

Flip **Trust X-Forwarded-For** off on the demos page and run `xff-spoof` again. The
same 90 requests go from 0 blocked to 60 blocked, because the guard now uses the
socket peer instead of a header the client controls.

Other scenarios keep working in both modes: they claim identity over `X-Demo-Client-IP`,
honored only for a loopback caller presenting a token generated in memory at startup.
Only `xff-spoof` uses the untrusted header, because testing whether the server believes
it is the entire point.

### Adding your own

One entry in the `scenarios` slice in
[internal/demo/scenarios.go](internal/demo/scenarios.go). Nothing else needs wiring. It
shows up in the API listing and the UI automatically.

```go
var scenarios = []Scenario{
    // ...
    {
        Meta: Meta{
            ID:      "my-scenario",
            Name:    "My scenario",
            Summary: "One line for the card.",
            Teaches: "What this reveals about the guard.",
            Expect:  "What the operator should watch for.",
            Tags:    []string{"rate limiting"},
        },
        Run: func(ctx context.Context, c *Client) error {
            limit := c.Policy().Limit // scale to whatever is configured
            c.Note("Starting")

            for range limit * 2 {
                if _, err := c.Get(ctx, As("203.0.113.9"), "/api/accounts/1"); err != nil {
                    return err
                }
            }
            return nil
        },
    },
}
```

Scale request counts off `c.Policy().Limit` rather than hardcoding them, as every
existing scenario does. A scenario written for a limit of 30 demonstrates nothing once
somebody tunes the limit to 5 in the console.

`As(ip)` claims an identity over the trusted channel. `Spoof(ip)` sends a raw
`X-Forwarded-For`. Runs are capped at 400 requests and 90 seconds, one at a time.

**After changing a scenario, re-record the public demo** or the Pages build keeps
replaying the old behavior. The easiest way is Actions → **Record demo data** →
_Run workflow_. See [Regenerating the demo data](#regenerating-the-demo-data).

---

## GitHub Page Preview

GitHub Pages serves static files only. No Go process, no Postgres, no SSE, no demo
runner. So the published console runs in **replay mode**. `cmd/record` drives your
real local server, captures each scenario's genuine run output along with the
`security_events` rows it produced, and writes them to `web/public/replay/`. The
static build replays those recordings at a watchable speed.

Recording rather than simulating means there is no second implementation of the rate
limiter to drift from. The numbers on the public demo are the numbers the real guard
produced, and both client-IP modes are recorded, so the vulnerable/hardened toggle
still works.

Two things are degraded, and the banner says so on every page: ledger edits live in the
browser tab and vanish on reload, and the limiter policy is fixed at recording time.

**One-time setup:** in the repo settings, set **Pages → Source → GitHub Actions**. The
workflow cannot do this for you. After that,
[.github/workflows/pages.yml](.github/workflows/pages.yml) builds and deploys on every
push to `main`.

### Regenerating the demo data

Press the button: **Actions → Record demo data → Run workflow**.

[.github/workflows/record.yml](.github/workflows/record.yml) stands up Postgres, builds
and boots the API, seeds the ledger, runs all five scenarios in both client-IP modes,
commits the fresh fixtures back to `main`, and redeploys Pages. Nothing to install
locally. Two inputs:

- **fake_seed** makes the fake ledger reproducible — the same value always gives the
  same 50 accounts, so leave it alone unless you want different names.
- **deploy** can be unticked to record and commit without publishing.

**Re-running with no code changes produces no commit.** The recorder diffs against the
committed fixture ignoring timestamps and durations, so the millisecond noise of a
fresh run does not count as a change. The timings stay in the file — they drive the
replay pacing — they just do not trigger a write.

To record and preview locally instead:

```bash
npm run record         # needs the API running
npm run build:pages
npm run preview:pages  # http://localhost:4173/Go-Ledger/
```

Stop the Go server before checking — serving with no backend at all is the real test.
That path comes from `VITE_BASE` in [web/.env.pages](web/.env.pages) and must match the
repository name, since Pages puts it in front of every asset URL; set it to `/` for a
user page or custom domain. The router base path and the fixture URLs both derive from
it.

One quirk worth not chasing: a deep link like `/Go-Ledger/demos` returns an HTTP 404
_status_ while rendering correctly. Pages has no history fallback, so the build copies
`index.html` to `404.html` and the SPA boots and routes on the real URL. Only the status
line is wrong.

---

## API

Lists share one envelope: `{"data": [...], "total": N, "limit": N, "offset": N}`.
Errors are always `{"error": "..."}`.

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

`amount` and `balance` are JSON **numbers**, not strings. `pgtype.Numeric` hands the
raw bytes to the Postgres numeric parser, so `"100.50"` is rejected as an invalid body.

### Console: `/ops`, not rate limited, no auth

| Method | Path                         | Notes                                                                                   |
| ------ | ---------------------------- | --------------------------------------------------------------------------------------- |
| GET    | `/ops/config`                | Active IP mode, limiter policy, and how the guard sees _you_                            |
| GET    | `/ops/events`                | `flag_status`, `ip_address` (comma-separated), `action_type`, `since`, `limit`, `offset` |
| GET    | `/ops/events/stream`         | SSE, honors `Last-Event-ID`, heartbeats every 15s                                       |
| GET    | `/ops/stats`                 | `window`, totals, top IPs, per-minute buckets, currently-blocked IPs                    |
| GET    | `/ops/demos`                 | The five scenarios and their metadata                                                   |
| POST   | `/ops/demos/{id}/run`        | Returns the full step list and a verdict                                                |
| PUT    | `/ops/config/client-ip-mode` | `{"mode": "xff-trust-all" \| "remote-addr"}`                                            |
| PUT    | `/ops/config/limiter-policy` | `{"limit": 30, "window": "10s", "block_period": "30s"}`                                 |
| POST   | `/ops/events/reset`          | Truncates `security_events`                                                             |

The three mutating routes exist only when `DEMOS_ENABLED=true`.

---

## Configuration

Every value has a working default except `DATABASE_URL`. The full list, with defaults
and the reasoning behind each, lives in [.env.example](.env.example) — copy it to `.env`
and edit. It is the only copy on purpose: this section used to restate the table and had
already drifted out of date against
[internal/config/config.go](internal/config/config.go), which is where the defaults
actually come from.

Two worth knowing before you deploy anything: `CLIENT_IP_MODE` defaults to
`xff-trust-all`, which is the deliberate vulnerability, and `OPS_ENABLED` /
`DEMOS_ENABLED` default to `true` and have no authentication behind them. See
[SECURITY.md](SECURITY.md).

CLI: `go run ./cmd/server seed`, `... reset`, `... healthcheck`.

---

## Layout

```
cmd/server/          entrypoint, lifecycle, graceful shutdown
cmd/record/          captures real demo runs into the Pages replay fixtures
internal/
  api/               chi router, ledger handlers, DTOs
  config/            all environment parsing, one place
  db/                embedded SQL under migrations/, run automatically on boot
  demo/              scenario list, loopback client, runner
  httpx/             the shared JSON response envelope
  ops/               limiter, guard, IP resolver, recorder, hub, SSE, console
  seed/              gofakeit data
  spa/               serves the built console (embedded or from disk)
web/                 React console
  src/features/<f>/  <f>.api.ts (transport) · <f>.queries.ts (hooks) · <f>.types.ts (DTOs)
  src/routes/        TanStack Router, file-based
  src/lib/           fetch client, error normalizer, query keys, formatting
  src/replay/        static-build transport: recorded fixtures instead of a server
  src/theme/         ESTRAL palette, component overrides
  public/replay/     the recordings themselves (committed)
```

Replay mode has two _transport_ seams, which is the whole reason it is cheap: the
`request()` chokepoint in `src/lib/http/client.ts`, and the `EventSource` in
`src/features/security/useEventStream.tsx`. No feature module, query hook or table knows
which build it is running in.

It is read in three more files, but only to choose wording — "recording" instead of
"live" — which is left inline deliberately: a mode-copy module would read worse.

The frontend layering is deliberate and worth keeping: components import
`*.queries.ts`, never `*.api.ts`. The API modules are pure transport with no React in
them, which is what makes the error normalizer and cache-key factory possible in one
place instead of scattered through components.

### Building a single binary

```bash
npm run build
```

Produces `bin/server` — `bin/server.exe` on Windows — with the console embedded.
`//go:embed` fails at compile time when `dist/` is missing, so the embed lives behind an
`embed_spa` build tag. That way `go build ./...` and `go test ./...` still work on a
fresh clone with no Node installed.

---

## License

[MIT](LICENSE). The deliberate `X-Forwarded-For` vulnerability is documented in
[SECURITY.md](SECURITY.md) — read that before lifting
[internal/ops/clientip.go](internal/ops/clientip.go) into anything real.
