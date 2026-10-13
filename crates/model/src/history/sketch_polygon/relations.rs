// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{LineRelation, PlanarConstraint};
pub(super) fn append(
    geometry: &mut Sketch2D,
    chain: &SketchPolylineCreationReport,
    circle: SketchEntityId,
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchConstraintId>, SketchPolygonCreationError> {
    let radial = chain
        .points
        .iter()
        .map(|point| PlanarConstraint::PointOnCircularCurve {
            point: *point,
            curve: circle,
        });
    let equal = chain.lines.windows(2).map(|pair| PlanarConstraint::Lines {
        lines: [pair[0], pair[1]],
        relation: LineRelation::EqualLength,
    });
    radial
        .chain(equal)
        .map(|relation| {
            let id = ids
                .allocate_sketch_constraint(chain.sketch)
                .map_err(SketchPolygonCreationError::Constraint)?;
            geometry
                .add_constraint(id, relation)
                .map_err(SketchPolygonCreationError::Sketch)?;
            Ok(id)
        })
        .collect()
}
