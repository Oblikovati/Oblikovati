// SPDX-License-Identifier: GPL-2.0-only

package step

import (
	"fmt"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/step/part21"
	"oblikovati.org/kernel/exchange/step/topomap"
	"oblikovati.org/kernel/topo"
)

// Reader imports AP203/214/242 STEP solids into kernel bodies. It satisfies
// exchange.BodyImporter (the owned seam) so the model layer depends on the
// abstraction, not on this struct. The zero value is ready to use.
//
// Example:
//
//	bodies, warns, err := step.Reader{}.ImportSolids(data, exchange.TranslationOptions{})
type Reader struct{}

// compile-time assertion that Reader is a BodyImporter.
var _ exchange.BodyImporter = Reader{}

// ImportSolids parses the file, resolves its length unit, and builds one body per
// MANIFOLD_SOLID_BREP (solid) and BREP_WITH_VOIDS/SHELL_BASED_SURFACE_MODEL
// (surface) it finds. Non-fatal issues are returned as warnings.
func (Reader) ImportSolids(data []byte, opts exchange.TranslationOptions) ([]*topo.Body, []string, error) {
	f, err := part21.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	mmPerUnit, warns := unitScale(f.Graph)
	bodies, bw, err := importAllBreps(f.Graph, opts.ImportScale(mmPerUnit), opts)
	if err != nil {
		return nil, nil, err
	}
	return bodies, append(warns, bw...), nil
}

// unitScale resolves the file→mm length scale, warning when no unit is declared.
func unitScale(g *part21.EntityGraph) (float64, []string) {
	scale, found, err := mmPerLengthUnit(g)
	if err != nil {
		return scale, []string{fmt.Sprintf("unit resolution failed (%v); assuming mm", err)}
	}
	if !found {
		return scale, []string{"no length unit declared; assuming millimeters"}
	}
	return scale, nil
}

// importAllBreps imports every solid and surface B-rep in the graph, reporting one progress tick
// per body and honouring a cancel between bodies (#1647).
func importAllBreps(g *part21.EntityGraph, scale float64, opts exchange.TranslationOptions) ([]*topo.Body, []string, error) {
	var bodies []*topo.Body
	var warns []string
	shells := solidShellRefs(g)
	for i, shellID := range shells {
		if err := opts.Report("solids", i, len(shells)); err != nil {
			return nil, nil, err
		}
		body, w, err := topomap.SolidFromShell(g, shellID, true, scale, importFeature(shellID))
		if err != nil {
			return nil, nil, err
		}
		bodies = append(bodies, body)
		warns = append(warns, w...)
	}
	return bodies, warns, nil
}

// solidShellRefs returns the CLOSED_SHELL ids of every MANIFOLD_SOLID_BREP in file
// order (the entry point for solid import).
func solidShellRefs(g *part21.EntityGraph) []int {
	var shells []int
	for _, brep := range g.EntitiesOfType("MANIFOLD_SOLID_BREP") {
		if id, err := lastRef(brep.Params); err == nil {
			shells = append(shells, id)
		}
	}
	return shells
}

// importFeature seeds a stable per-import lineage feature id from the shell id, so
// reference keys survive reopen (the imported-body persistent-naming root, plan §3).
func importFeature(shellID int) string {
	return fmt.Sprintf("import:step#%d", shellID)
}
