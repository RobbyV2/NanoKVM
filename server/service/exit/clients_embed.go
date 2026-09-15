package exit

import "embed"

// The four native clients, templated at request time by RenderClient. Exactly
// these names are embedded: clients/README.md and clients/testdata (the mock
// kvm, the host test driver, the Windows checklist) stay out of the binary.
//
//go:embed clients/client.sh clients/client.ps1 clients/client.pl clients/client.py
var clientFiles embed.FS
