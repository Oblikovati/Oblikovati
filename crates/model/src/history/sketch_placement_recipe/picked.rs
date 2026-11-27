// SPDX-License-Identifier: GPL-2.0-only
//! Resolve explicitly selected point identities before appending placement geometry.
use super::*;

pub(super) fn append_points(
    geometry: &mut Sketch2D,
    recipe: &SketchPlacementRecipe,
    ids: &mut RecipeReservations<'_>,
) -> Result<(Vec<SketchEntityId>, Option<Vec<SketchEntityId>>), SketchPlacementCreationError> {
    if recipe.picked_points.is_empty() {
        let fresh = append_defining_positions(geometry, &recipe.points, ids)
            .map_err(SketchPlacementCreationError::Primitive)?;
        return Ok((fresh, None));
    }
    validate_picks(geometry, recipe)?;
    let mut points = Vec::with_capacity(recipe.points.len());
    let mut fresh = Vec::new();
    for (index, position) in recipe.points.iter().enumerate() {
        let id = match recipe.picked_points.get(&index) {
            Some(id) => *id,
            None => {
                let created = append_defining_positions(geometry, &[*position], ids)
                    .map_err(SketchPlacementCreationError::Primitive)?;
                fresh.extend(&created);
                created[0]
            }
        };
        points.push(id);
    }
    Ok((points, Some(fresh)))
}

fn validate_picks(
    geometry: &Sketch2D,
    recipe: &SketchPlacementRecipe,
) -> Result<(), SketchPlacementCreationError> {
    for (&index, &id) in &recipe.picked_points {
        let expected = recipe.points.get(index).ok_or_else(|| {
            SketchError::Definition(format!(
                "picked point index {index}: expected an index below {}",
                recipe.points.len()
            ))
        })?;
        let actual = geometry.point(id)?.position;
        if &actual != expected {
            return Err(SketchError::Definition(format!(
                "picked point {id:?} at {actual:?}: expected recipe position {expected:?}"
            ))
            .into());
        }
    }
    Ok(())
}
