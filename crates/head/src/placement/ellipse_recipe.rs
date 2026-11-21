// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{SketchError, SketchPlacementRecipe};
pub(super) fn from_clicks(
    [center, major, through]: [[f64; 2]; 3],
) -> Result<SketchPlacementRecipe, SketchError> {
    let delta = [major[0] - center[0], major[1] - center[1]];
    let radius = delta[0].hypot(delta[1]);
    if !radius.is_finite() || radius <= 0. {
        return Err(SketchError::Definition(format!(
            "ellipse major endpoints {center:?}, {major:?}: expected a positive finite radius"
        )));
    }
    let axis = [delta[0] / radius, delta[1] / radius];
    // Go aa8297d1 EllipseTool: only the perpendicular cursor distance sets the minor radius.
    let offset = [through[0] - center[0], through[1] - center[1]];
    let minor = (-axis[1] * offset[0] + axis[0] * offset[1]).abs();
    SketchPlacementRecipe::ellipse(center, axis, [radius, minor])
}
