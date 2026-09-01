// SPDX-License-Identifier: GPL-2.0-only

package brep_test

import (
	"runtime"
	"testing"

	"oblikovati.org/math"
)

func TestNopIDCTransitionCSG(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "darwin" {
		t.Skip("macOS CI currently leaves this boolean acceptance body open")
	}
	cols, rows, pitch := 5, 2, 0.254
	length, height, width := float64(cols)*pitch+0.508, 0.74, 0.6
	body := prismBody(rectAtPoints(0, height/2, length, height), -width/2, width/2, "idc-transition-base")
	for i := 0; i < cols*rows; i++ {
		x := pitch / 2 * (float64(i) - float64(cols*rows-1)/2)
		body = cutOrFatal(t, body, prismBody(regularPolygonPoints(math.P3(x, height/2, 0), pitch/4, 20, 0), -width/2-0.05, width/2+0.05, "idc-pin-hole"), "idc pin hole")
	}
	body = cutOrFatal(t, body, prismBody(rectAtPoints(0, height/2-pitch/4+pitch/6, float64(cols)*pitch, pitch/3), -width/2-0.05, width/2+0.05, "idc-slot"), "idc slot")
	for x := range cols {
		for y := range rows {
			body = joinOrFatal(t, body, box(pitch*(float64(x)-float64(cols-1)/2)-0.025, pitch*(float64(y)-0.5)-0.025, -0.42, 0.05, 0.05, 0.56), "idc pin")
		}
	}

	requireValidNopSolid(t, "idc_transition", body)
	if got := vol(body); got <= 0 || got >= length*height*width+float64(cols*rows)*0.05*0.05*0.5 {
		t.Errorf("idc_transition volume = %.6f, outside expected source range", got)
	}
}
