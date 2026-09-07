package boolean

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
)

func TestProbeScaleSweepAfter(t *testing.T) {
	for _, s := range []float64{1e4, 1e3, 1, 1e-3, 1e-4} {
		slab, _ := brep.SolidBlock(math.P3(-1*math.Scalar(s), -1*math.Scalar(s), 0), math.P3(1*math.Scalar(s), 1*math.Scalar(s), 0.6*math.Scalar(s)), "slab")
		rod, _ := brep.SolidCylinder(math.P3(0, 0, -0.1*math.Scalar(s)), math.V3(0, 0, 1), 0.3*s, 0.8*s)
		body, err := brep.BooleanDiag(brep.Difference, slab, rod, nil)
		if err != nil {
			fmt.Printf("scale %g: brep err %v\n", s, err)
			continue
		}
		v := Validate(body)
		mesh, _ := tessellate.TessellateBody(body, DefaultQuality())
		fmt.Printf("scale %g: faces=%d valid=%v free=%d\n", s, len(body.Faces()), v.Valid, tessellate.WeldedFreeEdgeCount(mesh))
	}
}
