// SPDX-License-Identifier: GPL-2.0-only

package feature

import "oblikovati.org/api/contract"

// The derived-assembly and shrinkwrap features satisfy the public derive contract
// (ADR-0018): both pull a source assembly into the part and expose the same scalar
// surface (kind, link state, source version, break-link). Shrinkwrap is the simplified
// flavor of the same contract (M11-F06, #631/#716).
var (
	_ contract.DerivedAssemblyComponent = (*DerivedAssemblyComponent)(nil)
	_ contract.DerivedAssemblyComponent = (*ShrinkwrapComponent)(nil)
)
