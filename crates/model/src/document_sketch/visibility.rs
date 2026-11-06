// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe};
use crate::sketch::SketchError;

impl PlacedSketch {
    /// Read saved display state without requiring a successful solve; example: `sketch.visible()`.
    pub fn visible(&self) -> bool {
        self.visible
    }
}
impl DocumentSketches {
    /// Change display state while retaining geometry and verification; example: `sketches.set_visible(id, false)?`.
    pub fn set_visible(&mut self, id: DocumentSketchId, visible: bool) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.sketches[index].visible = visible;
        Ok(())
    }
}
pub(super) fn restore_visibility(
    sketches: &mut DocumentSketches,
    recipe: &PlacedSketchRecipe,
    version: u32,
) -> Result<(), SketchError> {
    if recipe.visible.is_some() && version < 5 {
        return Err(SketchError::Definition(format!(
            "sketch {:?} visibility {:?} in recipe version {version}: expected version 5 for saved visibility",
            recipe.id, recipe.visible
        )));
    }
    sketches.set_visible(recipe.id, recipe.visible.unwrap_or(true))
}
