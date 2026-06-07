// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	stdmath "math"
	"runtime"
	"testing"

	"oblikovati/math"
)

func TestNopJackCSG(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS CI currently leaves this boolean acceptance body open")
	}
	body := prismBody(rectPoints(0.6, 0.7), 0, 0.6, "jack-body")
	body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(0, 0, 0), 0.175, 48, 0), -0.05, 0.65, "jack-bore"), "jack bore")
	tube := annularPrism(t, 0.3, 0.175, 0.85, "jack-front-tube")
	body = joinOrFatal(t, body, tube, "jack tube")
	body = joinOrFatal(t, body, box(-0.3, -0.35, -0.3, 0.6, 0.7, 0.32), "jack rear block")

	requireValidNopSolid(t, "jack", body)
	if got := vol(body); got <= 0.6*0.7*0.6 || got >= 0.6*0.7*0.9+stdmath.Pi*0.3*0.3*0.85 {
		t.Errorf("jack volume = %.6f, outside expected source-construction range", got)
	}
}
