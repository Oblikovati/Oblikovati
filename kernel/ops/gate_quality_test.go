// SPDX-License-Identifier: GPL-2.0-only

package ops

// gateQuality names one tessellation quality a structural mesh gate is evaluated at.
type gateQuality struct {
	name string
	q    Quality
}

// gateQualities returns every quality a STRUCTURAL mesh invariant — 0 free (unpaired) edges, 0 fold
// edges, manifoldness — must hold at. Those invariants are exact and sampling-independent: they are
// properties of the MESHER, not of one faceting. A gate that asserts them at a single quality therefore
// tests one sampling.
//
// This is not a theoretical concern. #1510: the covering-space periodic mesher folded its seam ONLY at
// PropertyQuality — DefaultQuality's rim step (1/32) is wider than the ParamAt dead zone that collapsed
// consecutive rim samples onto one chart node, so at most one sample fell in it and no duplicate formed.
// The shipped regression ran at DefaultQuality alone and stayed GREEN over a body carrying 12 free edges
// and 3 fold edges at the quality every mass-property readout uses.
//
// Example:
//
//	for _, gq := range gateQualities() {
//		mesh, _ := tessellate.TessellateBody(b, gq.q)
//		if free := freeEdgeCount(mesh); free != 0 {
//			t.Errorf("%s quality: %d free edges; want 0 (watertight)", gq.name, free)
//		}
//	}
func gateQualities() []gateQuality {
	return []gateQuality{
		{"default", DefaultQuality()},
		{"property", PropertyQuality()},
	}
}

// certGateQualities adds the crossing-certification suites' own intermediate faceting (capCertQuality,
// chord 0.005 / angle 2°) to gateQualities, so those gates sample three samplings rather than the one
// their moment tolerances were calibrated at.
func certGateQualities() []gateQuality {
	return append(gateQualities(), gateQuality{"cert", capCertQuality()})
}
