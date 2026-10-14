// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{
    CircumcircleOptions, SketchCircleConstruction, SketchCircularPlacementCheck, SketchError,
    SketchPlacementRecipe,
};
pub(super) fn center_recipe(
    [center, start, end]: [[f64; 2]; 3],
    construction: &dyn SketchCircleConstruction,
) -> Result<SketchPlacementRecipe, SketchError> {
    let ccw = construction
        .arc_direction([center, start, end])
        .map_err(|cause| SketchError::Definition(cause.to_string()))?;
    SketchPlacementRecipe::arc(center, [start, end], ccw)
}
pub(super) fn circular_recipe(
    points: [[f64; 2]; 3],
    arc: bool,
    options: CircumcircleOptions,
    construction: &dyn SketchCircleConstruction,
) -> Result<SketchPlacementRecipe, SketchError> {
    let seed = construction
        .circumcircle(points, options)
        .map_err(|cause| SketchError::Definition(cause.to_string()))?;
    let mut recipe = if arc {
        SketchPlacementRecipe::arc(seed.center, [points[0], points[2]], seed.counter_clockwise)
    } else {
        SketchPlacementRecipe::circle(seed.center, seed.radius)
    }?;
    recipe.circular_checks.push(SketchCircularPlacementCheck {
        member: 0,
        points,
        options,
    });
    Ok(recipe)
}
pub(super) fn radial_recipe(
    [center, rim]: [[f64; 2]; 2],
) -> Result<SketchPlacementRecipe, SketchError> {
    let radius = (rim[0] - center[0]).hypot(rim[1] - center[1]);
    SketchPlacementRecipe::circle(center, radius)
}
