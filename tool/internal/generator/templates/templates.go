// Package templates provides embedded templates for code generation
package templates

import "embed"

//go:embed sqlc/*.tpl sqlc/*.sql.tpl
var SQLCTemplates embed.FS

//go:embed hertz/*.tpl
var HertzTemplates embed.FS

//go:embed kitex/*.tpl
var KitexTemplates embed.FS
