//go:build !embed_spa

package spa

import "io/fs"

// embedded reports no bundle. This is the default so that `go build ./...` and
// `go test ./...` work on a fresh clone without Node installed.
func embedded() (fs.FS, bool) { return nil, false }
