// SPDX-License-Identifier: GPL-2.0-only
module oblikovati.org/model/exchange/translators/solidworks

go 1.26.4

require (
	oblikovati.org v0.0.0
	oblikovati.org/model/exchange/translators/olecf v0.0.0
)

require (
	golang.org/x/image v0.0.0-20211028202545-6944b10bf410 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	oblikovati.org/api v0.144.0 // indirect
)

// Resolved locally via the workspace (go.work); the replaces let non-workspace tooling
// find the sibling checkouts. Paths are relative to this module dir
// (Oblikovati/model/exchange/translators/solidworks).
replace oblikovati.org => ../../../../

replace oblikovati.org/api => ../../../../../Oblikovati.API

replace oblikovati.org/model/exchange/translators/olecf => ../olecf
