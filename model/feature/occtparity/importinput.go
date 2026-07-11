// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"os"

	"oblikovati.org/kernel/exchange"
	stepio "oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// importInput loads OCCT's exact input solid (STEP-exported by the oracle) as our body, so
// the only variable under test downstream is our fillet, not shape construction.
//
// Example:
//
//	b, err := importInput("testdata/A1.step")
func importInput(stepPath string) (*topo.Body, error) {
	data, err := os.ReadFile(stepPath)
	if err != nil {
		return nil, fmt.Errorf("importInput: read %s: %w", stepPath, err)
	}
	bodies, _, err := stepio.Reader{}.ImportSolids(data, exchange.TranslationOptions{TargetUnitMM: 1})
	if err != nil {
		return nil, fmt.Errorf("importInput: import %s: %w", stepPath, err)
	}
	if len(bodies) != 1 {
		return nil, fmt.Errorf("importInput: %s produced %d bodies, want 1", stepPath, len(bodies))
	}
	return bodies[0], nil
}

// inputArea is the imported body's surface area, used to sanity-check STEP round-trip
// fidelity before blaming a fillet defect on our engine.
func inputArea(b *topo.Body) float64 {
	return ops.BodyGeometryProperties(b, ops.PropertyQuality()).Area
}
