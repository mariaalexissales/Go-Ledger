//go:build !embed_spa

package spa

import "io/fs"

func embedded() (fs.FS, bool) { return nil, false }
