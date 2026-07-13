// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

func TestObstacleProviderName(t *testing.T) {
	var p bsplineObstacleProvider
	if got := p.Name(); got != BlendKindBSpline {
		t.Errorf("obstacle provider Name() = %q, want %q", got, BlendKindBSpline)
	}
}

func TestObstacleRequestNilByDefault(t *testing.T) {
	req := CornerBlendRequest{}
	if req.ObstacleFeature != nil {
		t.Errorf("a default CornerBlendRequest must carry no ObstacleFeature (junction request unchanged)")
	}
}
