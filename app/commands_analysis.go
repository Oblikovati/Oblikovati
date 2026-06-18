// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/types"
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
	density := 0.0 // default to the assigned material density
	if props, ok := s.PhysicalProperties(); ok && props.Density > 0 {
		density = props.Density
	}
	mp := analysis.MassPropertiesOf(part.SurfaceBodies().All(), density, types.MassPropertiesMedium)
	s.SetNotice(fmt.Sprintf("Physical Properties — volume %.2f mm³, area %.2f mm², mass %.2f g, CoM (%.2f, %.2f, %.2f) mm, principal I (%.1f, %.1f, %.1f) g·mm²",
		mp.VolumeMm3, mp.SurfaceAreaMm2, mp.MassG, mp.CentroidXMm, mp.CentroidYMm, mp.CentroidZMm,
		mp.PrincipalMomentsGmm2[0], mp.PrincipalMomentsGmm2[1], mp.PrincipalMomentsGmm2[2]))
	return nil
}
