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
    kind: CircularRecipeKind,
    options: CircumcircleOptions,
    construction: &dyn SketchCircleConstruction,
) -> Result<SketchPlacementRecipe, SketchError> {
    let seed = construction
        .circumcircle(points, options)
        .map_err(|cause| SketchError::Definition(cause.to_string()))?;
    let mut recipe = kind.build(seed, [points[0], points[2]])?;
    recipe.circular_checks.push(SketchCircularPlacementCheck {
        member: 0,
        cap_members: if matches!(kind, CircularRecipeKind::Slot { .. }) {
            Some([3, 1])
        } else {
            None
        },
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

#[derive(Clone, Copy)]
pub(super) enum CircularRecipeKind {
    Circle,
    Arc,
    Slot { width: f64 },
}
impl CircularRecipeKind {
    fn build(
        self,
        seed: oblikovati_model::sketch::PlanarCircumcircle,
        endpoints: [[f64; 2]; 2],
    ) -> Result<SketchPlacementRecipe, SketchError> {
        match self {
            Self::Circle => SketchPlacementRecipe::circle(seed.center, seed.radius),
            Self::Arc => SketchPlacementRecipe::arc(seed.center, endpoints, seed.counter_clockwise),
            Self::Slot { width } => SketchPlacementRecipe::arc_slot(
                seed.center,
                endpoints,
                width,
                seed.counter_clockwise,
            ),
        }
    }
}
pub(super) fn center_slot_recipe(
    points: [[f64; 2]; 3],
    width: f64,
    construction: &dyn SketchCircleConstruction,
) -> Result<SketchPlacementRecipe, SketchError> {
    let ccw = construction
        .arc_direction(points)
        .map_err(|cause| SketchError::Definition(cause.to_string()))?;
    SketchPlacementRecipe::arc_slot(points[0], [points[1], points[2]], width, ccw)
}
