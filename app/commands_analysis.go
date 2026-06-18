// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/model/analysis"
)

// Analysis ribbon commands (M18-F01 #423): the Inspect tab's measurement and properties tools.
// Physical Properties computes the active part's mass properties and reports them in the status bar.

// analysisCommands are the Inspect tab's Measure panel.
func analysisCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Inspect.PhysicalProperties", "Physical Properties", "Measure", physicalProperties).
			WithTab("Inspect").WithRibbons(PartRibbon).WithEnable(hasActivePart).
			WithIcon("physical-properties").WithButtonStyle(LargeIconButton).
			WithTooltip("Compute the part's volume, surface area, centre of mass and mass, and show them in the status bar."),
	}
}

// physicalProperties computes the active part's mass properties and reports them as a notice.
func physicalProperties(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	mp := analysis.MassPropertiesOf(part.SurfaceBodies().All(), 0)
	s.SetNotice(fmt.Sprintf("Physical Properties — volume %.2f mm³, area %.2f mm², mass %.2f g, CoM (%.2f, %.2f, %.2f) mm",
		mp.VolumeMm3, mp.SurfaceAreaMm2, mp.MassG, mp.CentroidXMm, mp.CentroidYMm, mp.CentroidZMm))
	return nil
}
