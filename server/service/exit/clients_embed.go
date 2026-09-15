package exit

import "embed"

// The native clients (client.sh, client.ps1, client.pl, client.py) are written
// by the clients package brief and templated at request time by RenderClient.
// The directory is embedded whole and looked up by name, so this package
// compiles before the clients land and serves whatever is present; a missing
// script is a 404 like every other rejection on the token-gated surface.
//
//go:embed all:clients
var clientFiles embed.FS
