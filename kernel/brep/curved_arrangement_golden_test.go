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
// It is the drift gate on the machinery every curved boolean shares — trimByImprint → assembleSegments →
// arrangeBand → keptCells → curvedStitch. Each fixture is one cut whose structure and naming must stay
// put through work on that core.
//
// REBASELINED for ADR-0061 stage 4. The nine cases used to call nine bespoke drivers
// (RuledCrossingCutGeneral, SteinmetzCutGeneral, the four cap-crossing slices, …); they now call Boolean,
// so what is pinned is the body the KERNEL ships rather than one an unreachable driver still built. Six
// of the nine tightened when they moved: E 5→4 and free 1→0, the retired drivers' seam edge — an edge
// used twice by ONE face — no longer emitted. Every Euler characteristic is unchanged (chi 0, 4 for the
// Steinmetz's two wedge shells, 2 for the partial penetration), so the topology class is the same body,
// and two-cap-tunnel is byte-identical.
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
	make       func() (*topo.Body, error)
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
		{"crossing-cylinder", "V4E4F4chi0free0", "V4E4F4chi0-74852750406b9602", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, cylZ(-6, 3, 12), cylX(-6, 1.5, 12))
		}},
		{"cone-cone-cut", "V4E4F4chi0free0", "V4E4F4chi0-311173ef8acd01bf", func() (*topo.Body, error) {
			fat, _ := brep.SolidCylinderCone(math.P3(0, 0, -6), math.P3(0, 0, 6), 2, 4, "fat")
			rod, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 0.8, 1.5, "rod")
			return brep.Boolean(brep.Difference, fat, rod)
		}},
		{"cone-cylinder-cut", "V4E4F4chi0free0", "V4E4F4chi0-02f753d9a7fecdd0", func() (*topo.Body, error) {
			cyl := cylZ(-6, 3, 12)
			cone, _ := brep.SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
			return brep.Boolean(brep.Difference, cyl, cone)
		}},
		// Re-baselined for the unified radial stitch (ADR-0058): the equal-radius Steinmetz cut leaves
		// TWO wedge components touching at the two pinch points; the radial vertex-disk split now
		// separates them into coincident-but-distinct shells (ADR-0047's bowtie rule, previously planar
		// only) — V6/chi4 (two chi-2 shells) instead of the old shared-pinch-vertex V4/chi2 complex.
		{"steinmetz-cut", "V6E6F6chi4free0", "V6E6F6chi4-6a96f933d9fae724", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, cylX(-6, 3, 12), cylZ(-6, 3, 12))
		}},
		{"partial-penetration", "V4E4F5chi2free0", "V4E4F5chi2-93bc743a5194060b", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, cylZ(-6, 3, 12), cylX(-6, 1.5, 6))
		}},
		// --- the four certified #1724 cap-crossing slices (ride ruledOperandOf dispatch) ---
		{"cap-crossing-interior", "V4E4F4chi0free0", "V4E4F4chi0-649f4b16c84d4e93", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, target(), oblique45(-6.5))
		}},
		{"rim-crossing", "V4E5F4chi0free0", "V4E5F4chi0-706e075703bed170", func() (*topo.Body, error) {
			return brep.Boolean(brep.Difference, target(), oblique45(-5.6))
		}},
		{"two-cap-tunnel", "V4E5F4chi0free1", "V4E5F4chi0-3e6d014bf261527b", func() (*topo.Body, error) {
			th := 20.0 * stdmath.Pi / 180
			ux, uz := stdmath.Sin(th), stdmath.Cos(th)
			tool, _ := brep.SolidCylinder(math.P3(-2.416, 0, -2.518), math.V3(math.Scalar(ux), 0, math.Scalar(uz)), 0.7, 16)
			return brep.Boolean(brep.Difference, target(), tool)
		}},
		{"cone-cap-crossing", "V4E4F4chi0free0", "V4E4F4chi0-da7299cbc9295d07", func() (*topo.Body, error) {
			s := 1 / stdmath.Sqrt2
			top := math.P3(math.Scalar(-6.5+16*s), 0, math.Scalar(2+16*s))
			tool, _ := brep.SolidCylinderCone(math.P3(-6.5, 0, 2), top, 0.9, 0.6, "cone")
			return brep.Boolean(brep.Difference, target(), tool)
		}},
	}
}

func TestCurvedArrangementGolden(t *testing.T) {
	t.Parallel()
	for _, c := range arrangementGoldenCases() {
		t.Run(c.name, func(t *testing.T) {
			b, err := c.make()
			if err != nil || b == nil {
				t.Fatalf("%s: boolean failed (err=%v nil=%v) — fixture no longer classifies", c.name, err, b == nil)
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
