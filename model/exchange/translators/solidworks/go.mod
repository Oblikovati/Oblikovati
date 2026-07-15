// SPDX-License-Identifier: GPL-2.0-only
module oblikovati.org/model/exchange/translators/solidworks

go 1.26.4

require (
	oblikovati.org v0.0.0
	oblikovati.org/model/exchange/translators/olecf v0.0.0
)

// Resolved locally via the workspace (go.work); the replaces let non-workspace tooling
// find the sibling checkouts. Paths are relative to this module dir
// (Oblikovati/model/exchange/translators/solidworks).
replace oblikovati.org => ../../../../

replace oblikovati.org/api => ../../../../../Oblikovati.API

replace oblikovati.org/model/exchange/translators/olecf => ../olecf
