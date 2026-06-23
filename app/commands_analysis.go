// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/model/analysis"
)

// Analysis ribbon commands (M18-F01 #423): the Inspect tab's measurement and properties tools.
// Physical Properties computes the active part's mass properties and reports them in the status bar.

// analysisCommands are the Inspect tab's Measure panel.
func analysisCommands() []*CommandDefinition {
	return []*CommandDefinition{
		NewCommand("Inspect.Measure", "Measure", "Measure", startMeasure).
			WithTab("Inspect").WithRibbons(PartRibbon).WithEnable(hasActivePart).
			WithIcon("measure").WithButtonStyle(LargeIconButton).
			WithTooltip("Pick faces, edges and vertices to measure length, area, perimeter, distance and angle."),
		NewCommand("Inspect.PhysicalProperties", "Physical Properties", "Measure", physicalProperties).
			WithTab("Inspect").WithRibbons(PartRibbon).WithEnable(hasActivePart).
			WithIcon("physical-properties").WithButtonStyle(LargeIconButton).
			WithTooltip("Compute the part's volume, surface area, centre of mass and mass, and show them in the status bar."),
		NewCommand("Inspect.ModelHealth", "Model Health", "Validate", modelHealth).
			WithTab("Inspect").WithRibbons(PartRibbon).WithEnable(hasActivePart).
			WithIcon("model-health").WithButtonStyle(LargeIconButton).
			WithTooltip("Report the part's overall health and list every feature that is sick, warning or suppressed for repair."),
		NewCommand("Inspect.SurfaceAnalysis", "Surface Analysis", "Analyze", startSurfaceAnalysis).
			WithTab("Inspect").WithRibbons(PartRibbon).WithEnable(hasActivePart).
			WithIcon("surface-analysis").WithButtonStyle(LargeIconButton).
			WithTooltip("Surface Analysis — overlay reflection, highlight and isophote lines to judge surface fairness and G1/G2 continuity."),
		NewCommand("Inspect.Continuity", "Continuity Check", "Analyze", startContinuityCheck).
			WithTab("Inspect").WithRibbons(PartRibbon).WithEnable(hasActivePart).
			WithIcon("continuity-check").WithButtonStyle(LargeIconButton).
			WithTooltip("Continuity Check — pick a shared edge to report the G0 gap, G1 normal angle and G2 curvature difference across it."),
	}
}

// startContinuityCheck activates the cross-edge continuity checker on the active part.
func startContinuityCheck(s *Session) error {
	if _, err := activePart(s); err != nil {
		return err
	}
	s.StartTool(NewContinuityCheckTool())
	return nil
}

// startSurfaceAnalysis activates the live surface-interrogation overlay on the active part.
func startSurfaceAnalysis(s *Session) error {
	if _, err := activePart(s); err != nil {
		return err
	}
	s.StartTool(NewSurfaceInterrogationTool())
	return nil
}

// modelHealth aggregates the active part's feature health and reports it as a notice.
func modelHealth(s *Session) error {
	part, err := activePart(s)
	if err != nil {
		return err
	}
	mh := analysis.ModelHealthOf(part.Features())
	if len(mh.Unhealthy) == 0 {
		s.SetNotice("Model Health — all features OK")
		return nil
	}
	s.SetNotice(fmt.Sprintf("Model Health — overall %s, %d sick: %s",
		mh.Overall, mh.SickCount, unhealthySummary(mh.Unhealthy)))
	return nil
}

// unhealthySummary lists each unhealthy feature as "Name (status)".
func unhealthySummary(items []analysis.FeatureHealthItem) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = fmt.Sprintf("%s (%s)", it.Name, it.Status)
	}
	return strings.Join(parts, ", ")
}

// startMeasure activates the interactive Measure tool on the active part.
func startMeasure(s *Session) error {
	if _, err := activePart(s); err != nil {
		return err
	}
	s.StartTool(NewMeasureTool())
	return nil
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
