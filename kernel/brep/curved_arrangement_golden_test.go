// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"runtime"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Characterization golden for the shared curved-boolean (u,v)-arrangement core (#1732, Slice 0).
//
// This is the REGRESSION GATE for the partial-rim side-trim work (issue Oblikovati/Oblikovati#1732):
// composing a prior section-arc boundary with a new cut requires generalizing the arrangement machinery
// EVERY curved boolean shares — trimByImprint → assembleSegments → arrangeBand → keptCells — plus adding
// cutCylinderOperand to the ruledOperandOf dispatch. Both changes touch code the four certified #1724
// cap-crossing slices and the whole ruled/solid cut family depend on. This golden captures their output
// on develop BEFORE any change so the bare-band path stays BYTE-IDENTICAL through the refactor: the new
// operand and the constraint-edge ingest must be strictly ADDITIVE, engaged only for an already-cut
// target, so every fixture here — none of which is partial-rim — must come out unchanged.
//
// TWO TIERS, because the fixtures are faceted (cos/sin coordinates) and this job runs on ubuntu, macOS AND
// Windows (ci.yml Tier-1 matrix):
//   - structSig (V/E/F/chi/free-edge count) is integer-valued and platform-STABLE — the cross-platform hard
//     gate that catches a lost/gained cell, a merged face, or a torn boundary anywhere.
//   - sewSignature (the SHA over sorted reference keys, ADR-0043 naming) is coordinate-fragile: a ULP-level
//     divergence in a faceted rim can flip a same-parent rank and thus a key, the exact sensitivity that
//     failed a macOS golden in #1726. It is the byte-identical NAMING-drift gate and runs on Linux ONLY,
//     where the capture was taken; it is skipped elsewhere rather than allowed to flake CI red.
//
// sewSignature is shared with boolean_sew_golden_test.go (same brep_test package).

// structSig is the platform-stable structural fingerprint: counts only, no coordinates.
func structSig(b *topo.Body) string {
	free := 0
	for _, e := range b.Edges() {
		if len(e.Faces()) != 2 {
			free++
		}
	}
	return sprintfSig(len(b.Vertices()), len(b.Edges()), len(b.Faces()), b.EulerCharacteristic(), free)
}

func sprintfSig(v, e, f, chi, free int) string {
	return "V" + itoa(v) + "E" + itoa(e) + "F" + itoa(f) + "chi" + itoa(chi) + "free" + itoa(free)
}

// itoa avoids fmt in the hot signature path and keeps the format explicit (negative chi allowed).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// arrangementGoldenCase is one curved boolean whose exact output must survive the #1732 refactor.
type arrangementGoldenCase struct {
	name       string
	wantStruct string // asserted on every OS
	wantKeys   string // asserted on Linux only (reference-key SHA)
	make       func() (*topo.Body, bool)
}

func cylZ(z, r, h float64) *topo.Body {
	b, _ := brep.SolidCylinder(math.P3(0, 0, math.Scalar(z)), math.V3(0, 0, 1), math.Scalar(r), math.Scalar(h))
	return b
}

func cylX(x, r, h float64) *topo.Body {
	b, _ := brep.SolidCylinder(math.P3(math.Scalar(x), 0, 0), math.V3(1, 0, 0), math.Scalar(r), math.Scalar(h))
	return b
}

// oblique45 builds the certified 45° cylinder tool (r0.9, 16-gon) at the given base x — slices 1/2's fixture.
func oblique45(baseX float64) *topo.Body {
	s := 1 / stdmath.Sqrt2
	b, _ := brep.SolidCylinder(math.P3(math.Scalar(baseX), 0, 2), math.V3(math.Scalar(s), 0, math.Scalar(s)), 0.9, 16)
	return b
}

func arrangementGoldenCases() []arrangementGoldenCase {
	target := func() *topo.Body { return cylZ(0, 3, 10) } // the r=3 h=10 cap-crossing target
	return []arrangementGoldenCase{
		// --- solid-cut family (rides trimByImprint + ruledOperandOf directly) ---
		{"crossing-cylinder", "V4E5F4chi0free1", "V4E5F4chi0-daf5165df038e441", func() (*topo.Body, bool) {
			return brep.RuledCrossingCutGeneral(cylZ(-6, 3, 12), cylX(-6, 1.5, 12), nil)
		}},
		{"cone-cone-cut", "V4E5F4chi0free1", "V4E5F4chi0-62b46082d3a6e2e7", func() (*topo.Body, bool) {
			fat, _ := brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
			rod, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "rod")
			return brep.RuledCrossingCutGeneral(fat, rod, nil)
		}},
		{"cone-cylinder-cut", "V4E5F4chi0free1", "V4E5F4chi0-1f6dd86ad3e40a48", func() (*topo.Body, bool) {
			cyl := cylZ(-6, 3, 12)
			cone, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
			return brep.RuledCrossingCutGeneral(cyl, cone, nil)
		}},
		{"steinmetz-cut", "V4E6F6chi2free0", "V4E6F6chi2-9ebec3decb26122b", func() (*topo.Body, bool) {
			return brep.SteinmetzCutGeneral(cylX(-6, 3, 12), cylZ(-6, 3, 12), nil)
		}},
		{"partial-penetration", "V4E5F5chi2free1", "V4E5F5chi2-324d24b080bb6f9b", func() (*topo.Body, bool) {
			return brep.PartialPenetrationCutGeneral(cylZ(-6, 3, 12), cylX(-6, 1.5, 6), nil)
		}},
		// --- the four certified #1724 cap-crossing slices (ride ruledOperandOf dispatch) ---
		{"cap-crossing-interior", "V4E5F4chi0free1", "V4E5F4chi0-5970beadaf185034", func() (*topo.Body, bool) {
			return brep.CapCrossingCutGeneral(target(), oblique45(-6.5), nil)
		}},
		{"rim-crossing", "V4E5F4chi0free0", "V4E5F4chi0-5015ac6e262cd9e0", func() (*topo.Body, bool) {
			return brep.RimCrossingCutGeneral(target(), oblique45(-5.6), nil)
		}},
		{"two-cap-tunnel", "V4E5F4chi0free1", "V4E5F4chi0-0adcb840d5945a7f", func() (*topo.Body, bool) {
			th := 20.0 * stdmath.Pi / 180
			ux, uz := stdmath.Sin(th), stdmath.Cos(th)
			tool, _ := brep.SolidCylinder(math.P3(-2.416, 0, -2.518), math.V3(math.Scalar(ux), 0, math.Scalar(uz)), 0.7, 16)
			return brep.TwoCapCrossingCutGeneral(target(), tool, nil)
		}},
		{"cone-cap-crossing", "V4E5F4chi0free1", "V4E5F4chi0-5308a224b7afb0c5", func() (*topo.Body, bool) {
			s := 1 / stdmath.Sqrt2
			top := math.P3(math.Scalar(-6.5+16*s), 0, math.Scalar(2+16*s))
			tool, _ := brep.SolidCylinderCone(math.P3(-6.5, 0, 2), top, 0.9, 0.6, "cone")
			return brep.ConeCapCrossingCutGeneral(target(), tool, nil)
		}},
	}
}

func TestCurvedArrangementGolden(t *testing.T) {
	for _, c := range arrangementGoldenCases() {
		t.Run(c.name, func(t *testing.T) {
			b, ok := c.make()
			if !ok || b == nil {
				t.Fatalf("%s: builder declined (ok=%v nil=%v) — fixture no longer classifies", c.name, ok, b == nil)
			}
			gotStruct := structSig(b)
			if c.wantStruct == "" {
				t.Fatalf("%s: CAPTURE struct=%q keys=%q", c.name, gotStruct, sewSignature(b))
			}
			if gotStruct != c.wantStruct {
				t.Fatalf("%s: structural signature drifted\n  got  %s\n  want %s", c.name, gotStruct, c.wantStruct)
			}
			if runtime.GOOS != "linux" {
				return // reference-key hash is Linux-anchored (faceted-coordinate ULP fragility, #1726)
			}
			if got := sewSignature(b); got != c.wantKeys {
				t.Fatalf("%s: reference-key signature drifted (naming/ADR-0043)\n  got  %s\n  want %s", c.name, got, c.wantKeys)
			}
		})
	}
}
