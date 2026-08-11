# Security

**The vulnerability in this repo is deliberate.** The default client-IP resolver
trusts `X-Forwarded-For` verbatim, which makes the rate limiter trivially
bypassable. Demonstrating that is the entire point of the project — the
`xff-spoof` demo scenario exists to exploit it, and the recorded fixtures behind
the published console show it succeeding. It is not a bug and does not need
reporting.

Two things to know if you are reading this because a scanner flagged something:

- **`CLIENT_IP_MODE`** defaults to `xff-trust-all`. Set it to `remote-addr` to
  close the header hole. See the note in `internal/ops/clientip.go`.
- **`/ops` and `/demos` have no authentication.** Set `OPS_ENABLED=false` and
  `DEMOS_ENABLED=false` anywhere that is not a local demo.

Nothing here is intended for production, and no version of it is supported.

## Reporting something else

If you find a flaw that is *not* the one above — a way to reach `/ops` with it
disabled, an injection in the ledger handlers, a path traversal in the SPA
handler — open a regular GitHub issue. There is no private disclosure process
and no expectation of one: this repo holds no data and nobody depends on it.
