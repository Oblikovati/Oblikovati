// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchAxisOperand;
pub(super) fn pair(
    kind: &GeometricConstraintKind,
    first: &Operand,
    second: &Operand,
    request: &AddConstraintArgs,
) -> Option<Creation> {
    let axes = [
        axis(
            first,
            selected_major_axis(&request.use_ellipse_one_major_axis),
        )?,
        axis(
            second,
            selected_major_axis(&request.use_ellipse_two_major_axis),
        )?,
    ];
    pair_relation(kind, axes)
}
fn pair_relation(kind: &GeometricConstraintKind, axes: [SketchAxisOperand; 2]) -> Option<Creation> {
    if *kind == GEO_CONSTRAINT_COLLINEAR {
        return relation(AxisCollinear { axes });
    }
    let perpendicular = match kind {
        kind if *kind == GEO_CONSTRAINT_PARALLEL => false,
        kind if *kind == GEO_CONSTRAINT_PERPENDICULAR => true,
        _ => return None,
    };
    relation(AxisRelation {
        axes,
        perpendicular,
    })
}
fn axis(operand: &Operand, major_axis: bool) -> Option<SketchAxisOperand> {
    match &operand.kind {
        kind if *kind == SKETCH_ENTITY_LINE => Some(SketchAxisOperand::Line { line: operand.id }),
        kind if *kind == SKETCH_ENTITY_ELLIPSE => Some(SketchAxisOperand::Ellipse {
            ellipse: operand.id,
            major_axis,
        }),
        _ => None,
    }
}
