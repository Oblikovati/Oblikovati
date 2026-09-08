package tessellate_test

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
)

func TestZZWhichFaces(t *testing.T) {
	for _, name := range []string{"int", "join", "cut"} {
		op := map[string]ops.PartFeatureOperation{"int": ops.Intersect, "join": ops.Join, "cut": ops.Cut}[name]
		body := rodBall(t, op)
		for i, f := range body.Faces() {
			if _, planar := f.Geometry().(geom.Plane); planar {
				continue
			}
			t.Logf("%s face %d %T loops=%d chart=%d -> %s", name, i, f.Geometry(), len(f.Loops()),
				len(f.Chart()), tessellate.ClassifyCurvedTrimName(f, ops.DefaultQuality()))
		}
	}
}
