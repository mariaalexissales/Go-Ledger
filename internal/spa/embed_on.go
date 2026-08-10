//go:build embed_spa

package spa

import (
	"embed"
	"io/fs"
)

// The `all:` prefix is required or files Vite emits with a leading dot are
// skipped. dist/ is produced by `npm run build` in web/ and is gitignored.
//
//go:embed all:dist
var distFS embed.FS

func embedded() (fs.FS, bool) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, false
	}

	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}

	return sub, true
}
