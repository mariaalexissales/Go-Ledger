/**
 * Replay mode ships the console as a static site with no backend. Every
 * scenario is a recording of a real run against the Go server, captured by
 * `go run ./cmd/record`.
 */
export const REPLAY = import.meta.env.VITE_REPLAY === '1'
