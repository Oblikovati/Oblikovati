// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::SketchPlacementRelation;
pub(super) fn append(
    geometry: &mut Sketch2D,
    chain: &SketchPolylineCreationReport,
    circle: SketchEntityId,
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchConstraintId>, SketchPolygonCreationError> {
    let mut members = chain.lines.clone();
    members.push(circle);
    SketchPlacementRelation::polygon(chain.lines.len())
        .map(|relation| {
            let input = relation
                .resolve(&chain.points, &members)
                .map_err(SketchPolygonCreationError::Sketch)?;
            let id = ids
                .allocate_sketch_constraint(chain.sketch)
                .map_err(SketchPolygonCreationError::Constraint)?;
            input
                .append(geometry, id)
                .map_err(SketchPolygonCreationError::Sketch)?;
            Ok(id)
        })
        .collect()
}
