// Package frontend embute o build do React (npm run build) para o app desktop.
package frontend

import "embed"

// Dist é a pasta dist/ gerada pelo Vite. Rode npm run build antes de compilar.
//
//go:embed all:dist
var Dist embed.FS
