// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"fmt"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/param"
)

// Document-level parameter settings, tolerance sweeps, and parameter-set XML
// exchange over the wire (M02-F07, Oblikovati#606): parameters.getSettings/
// setSettings/setAllModelValueType/export/import.

// parameterSettingsInfo marshals the live settings into their wire DTO.
func parameterSettingsInfo(s *param.CollectionSettings) wire.ParameterSettingsInfo {
	return wire.ParameterSettingsInfo{
		LinearStandardTolerance:      s.LinearStandardTolerance,
		AngularStandardTolerance:     s.AngularStandardTolerance,
		UseStandardTolerances:        s.UseStandardTolerances,
		ExportStandardTolerances:     s.ExportStandardTolerances,
		LinearDimensionPrecision:     s.LinearDimensionPrecision,
		AngularDimensionPrecision:    s.AngularDimensionPrecision,
		DimensionDisplayType:         s.DimensionDisplayType.String(),
		DisplayParameterAsExpression: s.DisplayParameterAsExpression,
	}
}

// getParameterSettings returns the document's parameter settings.
func getParameterSettings(_ *app.Session, holder compdef.ParameterHolder) (wire.ParameterSettingsInfo, error) {
	return parameterSettingsInfo(holder.Parameters().Settings()), nil
}

// setParameterSettings applies the non-nil settings mutations, records one
// undo step, and returns the updated settings.
func setParameterSettings(s *app.Session, holder compdef.ParameterHolder, in wire.ParameterSettingsUpdateArgs) (wire.ParameterSettingsInfo, error) {
	settings := holder.Parameters().Settings()
	if err := applySettingsUpdate(settings, in); err != nil {
		return wire.ParameterSettingsInfo{}, err
	}
	s.RecordActiveEdit("Edit Parameter Settings")
	return parameterSettingsInfo(settings), nil
}

// applySettingsUpdate copies the non-nil update fields onto the settings,
// validating spellings and precision shape.
func applySettingsUpdate(s *param.CollectionSettings, in wire.ParameterSettingsUpdateArgs) error {
	setIfPresent(in.LinearStandardTolerance, &s.LinearStandardTolerance)
	setIfPresent(in.AngularStandardTolerance, &s.AngularStandardTolerance)
	setIfPresent(in.UseStandardTolerances, &s.UseStandardTolerances)
	setIfPresent(in.ExportStandardTolerances, &s.ExportStandardTolerances)
	setIfPresent(in.DisplayParameterAsExpression, &s.DisplayParameterAsExpression)
	if err := applyPrecision(in.LinearDimensionPrecision, &s.LinearDimensionPrecision, "linear"); err != nil {
		return err
	}
	if err := applyPrecision(in.AngularDimensionPrecision, &s.AngularDimensionPrecision, "angular"); err != nil {
		return err
	}
	if in.DimensionDisplayType == nil {
		return nil
	}
	display, ok := types.ParseDimensionDisplayType(*in.DimensionDisplayType)
	if !ok {
		return fmt.Errorf("parameters.setSettings: unknown dimension display type %q (want value|name|expression|tolerance|preciseValue)", *in.DimensionDisplayType)
	}
	s.DimensionDisplayType = display
	return nil
}

// applyPrecision validates and applies one display precision (decimal places).
func applyPrecision(in *int, dst *int, which string) error {
	if in == nil {
		return nil
	}
	if *in < 0 || *in > 8 {
		return fmt.Errorf("parameters.setSettings: %s precision %d out of range; want 0–8 decimal places", which, *in)
	}
	*dst = *in
	return nil
}

// sweepParameterModelValues drives every toleranced parameter's model-value
// selection to one bound, recomputes (the consumed values move), and records
// one undo step.
func sweepParameterModelValues(s *app.Session, in wire.ParameterSweepArgs) (wire.ParameterSweepResult, error) {
	m, ok := types.ParseModelValueType(in.ModelValueType)
	if !ok {
		return wire.ParameterSweepResult{}, fmt.Errorf("parameters.setAllModelValueType: unknown model value type %q (want nominal|lower|upper|median)", in.ModelValueType)
	}
	holder, err := modelaccess.ActiveParameterHolder(s)
	if err != nil {
		return wire.ParameterSweepResult{}, err
	}
	affected, err := holder.Parameters().SetAllModelValueType(m)
	if err != nil {
		return wire.ParameterSweepResult{}, err
	}
	holder.RecomputeAfterChange()
	s.RecordActiveEdit("Sweep Tolerances")
	return wire.ParameterSweepResult{Affected: affected}, nil
}

// exportParameters renders the user parameters as the exchange XML (read-only).
func exportParameters(_ *app.Session, holder compdef.ParameterHolder) (wire.ParameterExportResult, error) {
	xml, err := holder.Parameters().ExportXML()
	if err != nil {
		return wire.ParameterExportResult{}, err
	}
	return wire.ParameterExportResult{XML: xml}, nil
}

// importParameters applies an exchange XML through the session's atomic
// import seam (snapshot rollback on a bad set, one undo step on success).
func importParameters(s *app.Session, in wire.ParameterImportArgs) (wire.ParameterImportResult, error) {
	added, updated, err := s.ImportParameters(in.XML)
	if err != nil {
		return wire.ParameterImportResult{}, err
	}
	return wire.ParameterImportResult{Added: added, Updated: updated}, nil
}
