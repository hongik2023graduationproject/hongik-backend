// Package migrations bundles the SQL files into the binary so deployments
// never depend on a sidecar volume or a manually-applied script.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
