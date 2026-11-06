// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::*;
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
use oblikovati_session::selection::{SketchPick, SketchPickTarget};

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum OperandCategory {
    Point,
    Line,
    Circular,
}
#[derive(Clone, Copy, Debug)]
pub(crate) struct SketchOperand {
    pub id: SketchEntityId,
    pub category: OperandCategory,
}
pub(crate) fn category(kind: &SketchEntityKind) -> Option<OperandCategory> {
    if *kind == SKETCH_ENTITY_POINT {
        return Some(OperandCategory::Point);
    }
    if *kind == SKETCH_ENTITY_LINE {
        return Some(OperandCategory::Line);
    }
    if *kind == SKETCH_ENTITY_CIRCLE || *kind == SKETCH_ENTITY_ARC {
        return Some(OperandCategory::Circular);
    }
    None
}
pub(crate) fn resolve(
    sketch: &Sketch2D,
    picks: &[SketchPick],
) -> Result<Vec<SketchOperand>, String> {
    picks
        .iter()
        .map(|pick| resolve_pick(sketch, *pick))
        .collect()
}
fn resolve_pick(sketch: &Sketch2D, pick: SketchPick) -> Result<SketchOperand, String> {
    let SketchPickTarget::Entity(id) = pick.target() else {
        return Err(format!(
            "constraint pick {:?}: expected an owned sketch entity",
            pick.target()
        ));
    };
    let kind = sketch.entity_kind(id).map_err(|cause| cause.to_string())?;
    let category = category(&kind).ok_or_else(|| {
        format!(
            "constraint entity {id:?} with kind {kind:?}: expected a point, line, circle or arc"
        )
    })?;
    Ok(SketchOperand { id, category })
}
pub(crate) fn ids(operands: &[SketchOperand], category: OperandCategory) -> Vec<SketchEntityId> {
    operands
        .iter()
        .filter(|operand| operand.category == category)
        .map(|operand| operand.id)
        .collect()
}
