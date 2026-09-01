// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestShadowValidateAgainstExistingEngine runs the mesh-arrangement boolean beside
// the existing engine over representative operand pairs (planar boxes, crossing
// cylinders, a drilled block, a box/sphere) for every operation, and checks the new
// engine is a strict superset: each result is a valid closed solid whose volume
// matches the existing engine within the facet bias of the tessellated input. This
// is the ADR-0052 cutover gate — the new engine becomes the default only once it
// clears it. Skipped in -short (it runs both engines on curved solids).
func TestShadowValidateAgainstExistingEngine(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("shadow validation runs both boolean engines on curved solids")
	}
	q := DefaultQuality()
	// Operands are rebuilt per operation: the existing Boolean consumes its operand
	// bodies, so a body may not be reused across Join/Cut/Intersect.
	pairs := []struct {
		name string
		a, b func(t *testing.T) *topo.Body
		// truth overrides the existing engine as the oracle where the existing engine
		// is provably wrong (see box-sphere/intersect below). Value is the analytic /
		// numerically-integrated volume the new engine must reproduce within the facet
		// budget; the existing engine's answer is still logged for the record.
		truth map[PartFeatureOperation]float64
	}{
		{"boxes",
			func(*testing.T) *topo.Body { return block(0, 0, 0, 2, 2, 2) },
			func(*testing.T) *topo.Body { return block(1, 1, 1, 3, 3, 3) }, nil},
		{"crossing-cylinders",
			func(t *testing.T) *topo.Body { return mustCyl(t, math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12) },
			func(t *testing.T) *topo.Body { return mustCyl(t, math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12) }, nil},
		{"drilled-block",
			func(*testing.T) *topo.Body { return block(-2, -2, -2, 2, 2, 2) },
			func(t *testing.T) *topo.Body { return mustCyl(t, math.P3(0, 0, -3), math.V3(0, 0, 1), 1, 6) }, nil},
		// box-sphere/intersect: the existing engine returns 4.06, but the true box∩sphere
		// is 11.01 (sphere r=1.5 at (1,1,1) is almost wholly inside [-2,2]³; only three
		// small caps poke out). The existing engine is defective here; the new engine is
		// gated against the true value instead. This divergence motivates the cutover.
		{"box-sphere",
			func(*testing.T) *topo.Body { return block(-2, -2, -2, 2, 2, 2) },
			func(t *testing.T) *topo.Body { return mustSphere(t, math.P3(1, 1, 1), 1.5) },
			map[PartFeatureOperation]float64{Intersect: 11.0106}},
	}
	for _, pc := range pairs {
		for _, op := range []PartFeatureOperation{Join, Cut, Intersect} {
			existing, err := Boolean(op, pc.a(t), pc.b(t))
			if err != nil {
				t.Logf("%s/%s: existing engine declined (%v); skipping comparison", pc.name, op, err)
				continue
			}
			mop, _ := toMeshboolOp(op)
			mine := booleanViaMeshbool(pc.a(t), pc.b(t), mop, q, "shadow")
			if !mine.IsSolid() {
				t.Errorf("%s/%s: mesh-arrangement result is not a solid", pc.name, op)
				continue
			}
			if r := Validate(mine); !r.Valid {
				t.Errorf("%s/%s: mesh-arrangement result invalid: %+v", pc.name, op, r)
				continue
			}
			ve := query.BodyGeometryProperties(existing, q).Volume
			vm := query.BodyGeometryProperties(mine, q).Volume
			oracle, source := ve, "existing"
			if pc.truth != nil {
				if tv, ok := pc.truth[op]; ok {
					oracle, source = tv, "truth"
					t.Logf("%s/%s: NOTE existing engine returns %.4f, true value is %.4f (existing-engine defect)", pc.name, op, ve, tv)
				}
			}
			if rel := relDiff(oracle, vm); rel > 0.03 {
				t.Errorf("%s/%s: volume %.4f vs %s %.4f (%.2f%% off, > 3%% facet budget)", pc.name, op, vm, source, oracle, rel*100)
			} else {
				t.Logf("%s/%s: ok (mine=%.4f %s=%.4f, %.3f%%)", pc.name, op, vm, source, oracle, rel*100)
			}
		}
	}
}

func relDiff(a, b float64) float64 {
	d := stdmath.Abs(a - b)
	if m := stdmath.Max(stdmath.Abs(a), stdmath.Abs(b)); m > 0 {
		return d / m
	}
	return d
}

func block(x0, y0, z0, x1, y1, z1 float64) *topo.Body {
	b, _ := brep.SolidBlock(math.P3(x0, y0, z0), math.P3(x1, y1, z1), "block")
	return b
}

func mustCyl(t *testing.T, base math.Point3, axis math.Vector3, r, h float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidCylinder(base, axis, r, h)
	if err != nil {
		t.Fatalf("cylinder: %v", err)
	}
	return b
}

func mustSphere(t *testing.T, c math.Point3, r float64) *topo.Body {
	t.Helper()
	b, err := brep.SolidSphere(c, r, "sphere")
	if err != nil {
		t.Fatalf("sphere: %v", err)
	}
	return b
}
