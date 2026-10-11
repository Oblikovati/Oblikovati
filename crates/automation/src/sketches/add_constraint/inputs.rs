// SPDX-License-Identifier: GPL-2.0-only
use PlanarConstraint::*;
use oblikovati_api::{Error, types::*, wire::AddConstraintArgs};
use oblikovati_model::sketch::{
    LineRelation, PlanarConstraint, Sketch2D, SketchEntityId, SketchGeometricConstraintCreation,
};
type Creation = SketchGeometricConstraintCreation;
mod axes;
mod pairs;
#[derive(Clone)]
struct Operand {
    id: SketchEntityId,
    kind: SketchEntityKind,
}
pub(super) fn prepare(geometry: &Sketch2D, request: &AddConstraintArgs) -> Result<Creation, Error> {
    let refs = request.entities.as_deref().unwrap_or_default();
    let operands = refs
        .iter()
        .map(|id| operand(geometry, *id))
        .collect::<Result<Vec<_>, _>>()?;
    let kind = GeometricConstraintKind(request.kind.clone().into());
    let input = match operands.as_slice() {
        [single] => single_input(&kind, single, request),
        [first, second] => pairs::classify(&kind, first, second, request),
        _ => None,
    };
    input.ok_or_else(|| Error(format!(
        "geometric constructor {:?} operands {refs:?}: expected a supported canonical kind with owned constructor operands; this kind or operand shape is not ported", request.kind
    )))
}
fn operand(geometry: &Sketch2D, id: u64) -> Result<Operand, Error> {
    let id = SketchEntityId(id);
    let kind = geometry
        .entity_kind(id)
        .map_err(|cause| Error(cause.to_string()))?;
    Ok(Operand { id, kind })
}
fn single_input(
    kind: &GeometricConstraintKind,
    operand: &Operand,
    request: &AddConstraintArgs,
) -> Option<Creation> {
    if operand.kind == SKETCH_ENTITY_ELLIPSE {
        let major_axis = selected_major_axis(&request.use_ellipse_major_axis);
        return ellipse_alignment(kind, operand.id, major_axis);
    }
    point_line_input(kind, operand)
}
fn point_line_input(kind: &GeometricConstraintKind, operand: &Operand) -> Option<Creation> {
    match (&operand.kind, kind) {
        (shape, kind) if *shape == SKETCH_ENTITY_POINT && *kind == GEO_CONSTRAINT_FIX => {
            Some(Creation::Fix { point: operand.id })
        }
        (shape, kind) if *shape == SKETCH_ENTITY_LINE && *kind == GEO_CONSTRAINT_HORIZONTAL => {
            relation(LineHorizontal { line: operand.id })
        }
        (shape, kind) if *shape == SKETCH_ENTITY_LINE && *kind == GEO_CONSTRAINT_VERTICAL => {
            relation(LineVertical { line: operand.id })
        }
        _ => None,
    }
}
fn relation(relation: PlanarConstraint) -> Option<Creation> {
    Some(Creation::Relation(relation))
}
fn circular(kind: &SketchEntityKind) -> bool {
    *kind == SKETCH_ENTITY_CIRCLE || *kind == SKETCH_ENTITY_ARC
}

fn ellipse_alignment(
    kind: &GeometricConstraintKind,
    ellipse: SketchEntityId,
    major_axis: bool,
) -> Option<Creation> {
    let vertical = match kind {
        kind if *kind == GEO_CONSTRAINT_HORIZONTAL => false,
        kind if *kind == GEO_CONSTRAINT_VERTICAL => true,
        _ => return None,
    };
    relation(EllipseAxisAlignment {
        ellipse,
        major_axis,
        vertical,
    })
}

fn selected_major_axis(flag: &Option<std::boxed::Box<bool>>) -> bool {
    flag.as_deref().copied().unwrap_or(true)
}
