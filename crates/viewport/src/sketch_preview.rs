// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::sketch::{SketchPlacementRecipe, SketchPlane};
/// Owned transient geometry; permanent document identities never enter the renderer.
#[derive(Clone, Debug, PartialEq)]
pub struct ViewportSketchPreview {
    pub plane: SketchPlane,
    pub recipe: SketchPlacementRecipe,
}
