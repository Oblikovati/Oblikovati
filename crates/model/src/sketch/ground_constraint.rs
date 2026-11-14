// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchEntityId, SketchError, SketchPoint2D};
use oblikovati_solve::{ad::DualNumber, residual::ResidualError};
impl Sketch2D {
    pub(super) fn capture_ground_schema(&self, recipe: &mut super::SketchRecipe) {
        if !self
            .constraints
            .iter()
            .any(|constraint| matches!(constraint.relation, PlanarConstraint::Ground { .. }))
        {
            return;
        }
        recipe.version = 8;
        self.capture_complete_schema_metadata(recipe);
    }
}
pub(super) fn require_ground_schema(recipe: &super::SketchRecipe) -> Result<(), SketchError> {
    if recipe.version < 8
        && recipe
            .constraints
            .iter()
            .any(|constraint| matches!(constraint.relation, PlanarConstraint::Ground { .. }))
    {
        return Err(SketchError::Definition(format!(
            "ground recipe version {}: expected version 8 for frozen anchors",
            recipe.version
        )));
    }
    Ok(())
}

pub(super) fn capture_entity_ground(
    sketch: &Sketch2D,
    entity: SketchEntityId,
) -> Result<PlanarConstraint, SketchError> {
    let catalogue = super::entity_geometry::catalogue(sketch);
    let shape = catalogue.get(&entity).ok_or_else(|| {
        SketchError::Definition(format!(
            "ground entity {entity:?}: expected an owned sketch entity"
        ))
    })?;
    let points = shape
        .points()
        .into_iter()
        .map(|id| sketch.point(id).copied())
        .collect::<Result<Vec<_>, _>>()?;
    Ok(PlanarConstraint::Ground { points })
}
pub(super) fn validate_anchors(points: &[SketchPoint2D]) -> Result<(), SketchError> {
    let unique: std::collections::BTreeSet<SketchEntityId> =
        points.iter().map(|point| point.id).collect();
    if points.is_empty() || unique.len() != points.len() {
        return Err(SketchError::Definition(format!(
            "ground anchors {points:?}: expected a nonempty ordered list of distinct owned points"
        )));
    }
    points
        .iter()
        .try_for_each(|point| super::geometry::require_position(point.position))
}
pub(super) fn coordinate_residuals(
    variables: &[DualNumber],
    positions: &[[f64; 2]],
) -> Result<Vec<DualNumber>, ResidualError> {
    variables
        .iter()
        .zip(positions.iter().flatten())
        .map(|(coordinate, target)| coordinate.shifted(-target).map_err(ResidualError::from))
        .collect()
}
