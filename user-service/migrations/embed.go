package migrations

import "embed"

// Only embed .up.sql to avoid duplicate version (goose treats .up and .down as same version).
//go:embed *up.sql
var FS embed.FS
