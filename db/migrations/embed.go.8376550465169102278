// Package migrations bundles the SQL migration files into the binary via go:embed
// so the migration runner can apply them without depending on the filesystem layout.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
