package web

import "embed"

// Files are included in release builds by scripts/build-release.sh.
// The placeholder keeps ordinary development builds independent of a prior Vite build.
//
//go:embed static
var Files embed.FS
