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
    let mut used = BTreeSet::new();
    for lines in std::iter::once(&definition.selection.outer).chain(&definition.selection.holes) {
        validate_line_identities(lines, &mut used)?;
    }
    for (axis, source) in definition.displacement.into_iter().enumerate() {
        validate_scalar_source(source, axis)?;
    }
    if let DatumLength::Constant(normal) = definition.displacement[2] {
        require_normal_displacement(normal)?;
    }
    Ok(())
}
fn validate_line_identities(
    lines: &[SketchDirectedLine],
    used: &mut BTreeSet<SketchEntityId>,
) -> Result<(), SketchPrismCause> {
    if lines.len() < 3 {
        return Err(SketchPrismCause::Definition(format!(
            "sketch prism boundary {lines:?}: expected at least three directed lines"
        )));
    }
    for directed in lines {
        if directed.line.0 == 0 || !used.insert(directed.line) {
            return Err(SketchPrismCause::Definition(format!(
                "sketch prism line {:?}: expected nonzero identity with one occurrence across all boundaries",
                directed.line
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
