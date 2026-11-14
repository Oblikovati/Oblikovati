// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{LineRelation, PlanarConstraint, SketchDirectionInference};
use oblikovati_api::types::{
    INFER_HORIZONTAL, INFER_PARALLEL, INFER_PERPENDICULAR, INFER_VERTICAL,
};
pub(super) fn append(
    geometry: &mut Sketch2D,
    primitive: &SketchPrimitiveCreationReport,
    inference: &SketchLinePlacementInference,
    enabled: bool,
    ids: &mut RecipeReservations<'_>,
) -> Result<(Vec<SketchConstraintId>, Option<SketchAppliedLineDirection>), SketchLineCreationError>
{
    if !enabled {
        return Ok((Vec::new(), None));
    }
    let points = append_snaps(geometry, primitive, inference, ids)?;
    let direction = inference
        .direction
        .map(|suggestion| append_direction(geometry, primitive, suggestion, ids))
        .transpose()?;
    Ok((points, direction))
}

fn append_snaps(
    geometry: &mut Sketch2D,
    primitive: &SketchPrimitiveCreationReport,
    inference: &SketchLinePlacementInference,
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchConstraintId>, SketchLineCreationError> {
    let mut constraints = Vec::new();
    for snap in &inference.points {
        let relation = PlanarConstraint::Coincident {
            points: [primitive.points[snap.endpoint], snap.target],
        };
        constraints.push(append_relation(geometry, primitive.sketch, relation, ids)?);
    }
    Ok(constraints)
}
fn append_direction(
    geometry: &mut Sketch2D,
    primitive: &SketchPrimitiveCreationReport,
    inference: SketchDirectionInference,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchAppliedLineDirection, SketchLineCreationError> {
    let relation = direction_relation(primitive.entity, inference)?;
    let constraint = append_relation(geometry, primitive.sketch, relation, ids)?;
    let geometric_index = geometry
        .geometric_constraint_index(constraint)
        .map_err(SketchLineCreationError::Sketch)?;
    Ok(SketchAppliedLineDirection {
        inference,
        constraint,
        geometric_index,
    })
}
fn append_relation(
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    relation: PlanarConstraint,
    ids: &mut RecipeReservations<'_>,
) -> Result<SketchConstraintId, SketchLineCreationError> {
    let id = ids
        .allocate_sketch_constraint(sketch)
        .map_err(SketchLineCreationError::Constraint)?;
    geometry
        .add_constraint(id, relation)
        .map_err(SketchLineCreationError::Sketch)?;
    Ok(id)
}
fn direction_relation(
    line: crate::sketch::SketchEntityId,
    inference: SketchDirectionInference,
) -> Result<PlanarConstraint, SketchLineCreationError> {
    match (inference.kind, inference.target) {
        (INFER_HORIZONTAL, None) => Ok(PlanarConstraint::LineHorizontal { line }),
        (INFER_VERTICAL, None) => Ok(PlanarConstraint::LineVertical { line }),
        (INFER_PARALLEL, Some(target)) => Ok(PlanarConstraint::Lines {
            lines: [line, target],
            relation: LineRelation::Parallel,
        }),
        (INFER_PERPENDICULAR, Some(target)) => Ok(PlanarConstraint::Lines {
            lines: [line, target],
            relation: LineRelation::Perpendicular,
        }),
        shape => Err(SketchLineCreationError::Sketch(
            crate::sketch::SketchError::Definition(format!(
                "line direction inference {shape:?}: expected horizontal/vertical without a target or parallel/perpendicular with an owned line target"
            )),
        )),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn malformed_direction_shapes_keep_kind_and_target_in_the_decline() {
        let line = crate::sketch::SketchEntityId(1);
        for (kind, target) in [
            (INFER_HORIZONTAL, Some(line)),
            (INFER_VERTICAL, Some(line)),
            (INFER_PARALLEL, None),
            (INFER_PERPENDICULAR, None),
        ] {
            let inference = SketchDirectionInference {
                kind,
                target,
                score: 100,
            };
            let cause = direction_relation(line, inference).unwrap_err().to_string();
            assert!(cause.contains("expected horizontal/vertical"), "{cause}");
            assert!(cause.contains(&format!("{kind:?}")), "{cause}");
            assert!(cause.contains(&format!("{target:?}")), "{cause}");
        }
    }
}
