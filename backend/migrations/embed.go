// Package migrations embeds the goose SQL migrations so the `migrate` binary carries them (no files to copy
// onto the server). Files are NNNNN_<verb>_<thing>.sql; committed files are never edited (hook-enforced).
package migrations

import "embed"

// FS holds every migration file.
//
//go:embed *.sql
var FS embed.FS
