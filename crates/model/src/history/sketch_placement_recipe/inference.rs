// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{PlanarConstraint, SketchInferencePolicy};

pub(super) fn append(
    snapshot: &DocumentSnapshot,
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    fresh: &[SketchEntityId],
    ids: &mut RecipeReservations<'_>,
) -> Result<Vec<SketchConstraintId>, SketchPlacementCreationError> {
    let settings = snapshot.effective_sketch_settings();
    if !settings.auto_apply_constraints {
        return Ok(Vec::new());
    }
    let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
    let mut constraints = Vec::new();
    for point in fresh {
        let position = geometry.point(*point)?.position;
        let Some(target) = geometry.infer_point_coincidence(position, fresh, &settings, policy)?
        else {
            continue;
        };
        let constraint = ids
            .allocate_sketch_constraint(sketch)
            .map_err(SketchPlacementCreationError::Constraint)?;
        geometry.add_constraint(
            constraint,
            PlanarConstraint::Coincident {
                points: [*point, target],
            },
        )?;
        constraints.push(constraint);
    }
    Ok(constraints)
}
