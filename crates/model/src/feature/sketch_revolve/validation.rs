// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn validate_sources(
    definition: &SketchRevolveDefinition,
) -> Result<(), SketchRevolveCause> {
    if definition.sketch.0 == 0 {
        return Err(SketchRevolveCause::Definition(format!(
            "sketch {:?}: expected a nonzero source identity",
            definition.sketch
        )));
    }
    if let Some(axis) = definition.axis
        && axis.0 == 0
    {
        return Err(SketchRevolveCause::Definition(format!(
            "revolve axis {axis:?}: expected a nonzero sketch entity identity"
        )));
    }
    if definition.direction == SketchRevolveDirection::Asymmetric
        && definition.second_angle.is_none()
    {
        return Err(SketchRevolveCause::Definition(
            "revolve asymmetric direction: expected a second angle".into(),
        ));
    }
    crate::feature::sketch_prism::validation::validate_selection(&definition.selection)
        .map_err(|cause| SketchRevolveCause::Definition(cause.to_string()))?;
    validate_angle(definition.angle)?;
    if let Some(second) = definition.second_angle {
        validate_angle(second)?;
    }
    Ok(())
}

fn validate_angle(angle: DatumAngle) -> Result<(), SketchRevolveCause> {
    if angle.parameter_id() == Some(0) {
        return Err(SketchRevolveCause::Definition(format!(
            "revolve angle {angle:?}: expected a nonzero parameter identity"
        )));
    }
    angle.validate_source().map_err(SketchRevolveCause::Angle)
}
