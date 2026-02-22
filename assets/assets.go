// Package assets embeds all static files (SVG icons, UI).
package assets

import _ "embed"

//go:embed icon_on.svg
var IconOn string

//go:embed icon_off.svg
var IconOff string
