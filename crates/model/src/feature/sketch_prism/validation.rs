// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use std::collections::BTreeSet;
pub(super) fn validate_sources(definition: &SketchPrismDefinition) -> Result<(), SketchPrismCause> {
    if definition.sketch.0 == 0 {
        return Err(SketchPrismCause::Definition(format!(
            "sketch {:?}: expected a nonzero source identity",
            definition.sketch
        )));
    }
    validate_selection(&definition.selection)?;
    if definition.extent.is_some() {
        return validate_plane_extent(definition);
    }
    for (axis, source) in definition.displacement.into_iter().enumerate() {
        validate_scalar_source(source, axis)?;
    }
    if let DatumLength::Constant(normal) = definition.displacement[2] {
        definition.validate_normal_source(normal)?;
    }
    if (definition.direction == SketchPrismDirection::Asymmetric)
        != definition.second_depth.is_some()
    {
        return Err(SketchPrismCause::Definition(
            "asymmetric prism: expected Distance B only for a two-sided sweep".into(),
        ));
    }
    if let Some(depth) = definition.second_depth {
        validate_scalar_source(depth, 3)?;
        if let DatumLength::Constant(value) = depth {
            require_second_depth(value)?;
        }
    }
    Ok(())
}
pub(super) fn validate_plane_extent(
    definition: &SketchPrismDefinition,
) -> Result<(), SketchPrismCause> {
    if definition.displacement == [DatumLength::Constant(0.); 3]
        && definition.direction == SketchPrismDirection::Positive
        && definition.second_depth.is_none()
    {
        return Ok(());
    }
    Err(SketchPrismCause::Definition(
        "prism plane extent: expected plane bounds without authored distance or direction".into(),
    ))
}
pub(super) fn require_second_depth(value: f64) -> Result<(), SketchPrismCause> {
    if value <= 0. || !value.is_finite() {
        return Err(SketchPrismCause::Definition(format!(
            "prism Distance B {value:?}: expected a finite positive length"
        )));
    }
    Ok(())
}
/// Validate one shared directed closed-profile selection before a sweep operation.
/// Example: `validate_selection(&selection)?` before an authored revolve is accepted.
pub(crate) fn validate_selection(selection: &SketchPrismSelection) -> Result<(), SketchPrismCause> {
    let mut used = BTreeSet::new();
    for lines in std::iter::once(&selection.outer).chain(&selection.holes) {
        validate_curve_identities(lines, &mut used)?;
    }
    Ok(())
}
fn validate_curve_identities(
    curves: &[SketchDirectedCurve],
    used: &mut BTreeSet<SketchEntityId>,
) -> Result<(), SketchPrismCause> {
    if curves.is_empty() {
        return Err(SketchPrismCause::Definition(format!(
            "sketch prism boundary {curves:?}: expected at least one directed curve"
        )));
    }
    for directed in curves {
        if directed.curve.0 == 0 || !used.insert(directed.curve) {
            return Err(SketchPrismCause::Definition(format!(
                "sketch prism curve {:?}: expected nonzero identity with one occurrence across all boundaries",
                directed.curve
            )));
        }
    }
    Ok(())
}
fn validate_scalar_source(source: DatumLength, axis: usize) -> Result<(), SketchPrismCause> {
    if source.parameter() == Some(0) {
        return Err(SketchPrismCause::Definition(format!(
            "displacement axis {axis} source {source:?}: expected a nonzero parameter identity"
        )));
    }
    source
        .validate_source()
        .map_err(|cause| SketchPrismCause::Length {
            axis,
            source,
            cause,
        })
}
impl SketchPrismDefinition {
    pub(super) fn validate_normal_source(&self, normal: f64) -> Result<(), SketchPrismCause> {
        if self.direction != SketchPrismDirection::Positive && normal <= 0. {
            return Err(SketchPrismCause::Definition(format!(
                "prism {:?} depth {normal:?}: expected a positive authored distance",
                self.direction
            )));
        }
        if self.direction == SketchPrismDirection::Symmetric && normal / 2. == 0. {
            return Err(SketchPrismCause::Definition(format!(
                "symmetric prism depth {normal:?}: expected a representable half-distance"
            )));
        }
        if normal == 0. {
            return Err(SketchPrismCause::Definition(format!(
                "prism normal displacement {normal:?}: expected nonzero profile-normal displacement"
            )));
        }
        Ok(())
    }
}
