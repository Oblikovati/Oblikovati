// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "oblikovati.org/api/contract"

// The assembly feature program satisfies the public assembly-features contract
// (ADR-0018): the rollback marker and batch suppression the wire surface drives
// (M11-F08, #633/#725).
var (
	_ contract.AssemblyFeatures = (*AssemblyFeatures)(nil)
	_ contract.EndOfFeatures    = (*AssemblyFeatures)(nil)
)

// The assembly definition satisfies the public contract's scalar assembly read surface
// (#728); the occurrence tree and mutators travel over api/wire (assembly.*).
var _ contract.AssemblyComponentDefinition = (*AssemblyComponentDefinition)(nil)
