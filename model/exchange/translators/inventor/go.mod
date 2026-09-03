// SPDX-License-Identifier: GPL-2.0-only
module oblikovati.org/model/exchange/translators/inventor

go 1.27.0

require (
	github.com/klauspost/compress v1.18.0
	oblikovati.org v0.0.0
	oblikovati.org/api v0.154.0
	oblikovati.org/model/exchange/translators/olecf v0.0.0
)

require (
	golang.org/x/image v0.41.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Resolved locally via the workspace (go.work); the replaces let non-workspace tooling
// find the sibling checkouts. Paths are relative to this module dir
// (Oblikovati/model/exchange/translators/inventor).
replace oblikovati.org => ../../../../

replace oblikovati.org/api => ../../../../../Oblikovati.API

replace oblikovati.org/model/exchange/translators/olecf => ../olecf
