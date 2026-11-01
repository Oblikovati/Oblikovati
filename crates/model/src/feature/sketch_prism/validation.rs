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
    for (axis, source) in definition.displacement.into_iter().enumerate() {
        validate_scalar_source(source, axis)?;
    }
    if let DatumLength::Constant(normal) = definition.displacement[2] {
        require_normal_displacement(normal)?;
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
pub(super) fn require_normal_displacement(normal: f64) -> Result<(), SketchPrismCause> {
    if normal == 0. {
        return Err(SketchPrismCause::Definition(format!(
            "prism normal displacement {normal:?}: expected nonzero profile-normal displacement"
        )));
    }
    Ok(())
}
