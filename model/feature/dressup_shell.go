// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Dress-up features — the SHELL definition (M48 #2233 split of dressup.go). The hollow-out feature
// removing the selected faces to a wall thickness (with optional per-face thickness overrides) and its
// Recompute. The adder collection stays in dressup.go.

// ShellDefinition hollows a body, removing the selected faces, to a wall thickness.
type ShellDefinition struct {
	RemovedFaceKeys [][]byte
	Thickness       func() float64
	GeomFaces       []topo.GeometricFaceRef // externally-authored removed faces by geometric descriptor (ADR-0040)
	Direction       ops.ShellDirection      // which side the wall grows onto (default ShellInside) — #1864
	// FaceThicknesses give named RETAINED faces their own wall thickness (Inventor's
	// SetFaceThickness, #1864) — a thickened boss wall or a thin window in an otherwise uniform
	// shell. Each is a closure so a parameter drives it, exactly like the default Thickness.
	FaceThicknesses []ShellFaceThickness
}

// ShellFaceThickness overrides the wall thickness on one retained face of a shell (#1864).
type ShellFaceThickness struct {
	FaceKey   []byte
	Thickness func() float64
}

// ShellFeature hollows a solid.
type ShellFeature struct {
	def      *ShellDefinition
	featName string
}

func (s *ShellFeature) Definition() *ShellDefinition { return s.def }
func (s *ShellFeature) Kind() string                 { return "shell" }

// Recompute hollows the running body to the wall thickness, opening the removed faces. See
// shell.go.
func (s *ShellFeature) Recompute(in Input) (Output, error) {
	keys, err := bindGeomFaces(in, s.def.RemovedFaceKeys, s.def.GeomFaces, featOr(s.featName, "shell"))
	if err != nil {
		return Output{}, err
	}
	return shellBody(in, keys, callOrZero(s.def.Thickness), s.def.Direction,
		evalFaceThicknesses(s.def.FaceThicknesses), featOr(s.featName, "shell"))
}
