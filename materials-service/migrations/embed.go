package migrations

import "embed"

// Only .up.sql to avoid duplicate version in goose.
//go:embed *up.sql
var FS embed.FS
