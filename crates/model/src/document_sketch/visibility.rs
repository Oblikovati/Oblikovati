// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe};
use crate::sketch::SketchError;

impl PlacedSketch {
    /// Read saved display state without requiring a successful solve; example: `sketch.visible()`.
    pub fn visible(&self) -> bool {
        self.visible
    }
    /// Whether relations draw their marks; example: `sketch.constraints_shown()`.
    pub fn constraints_shown(&self) -> bool {
        self.constraints_shown
    }
}
impl DocumentSketches {
    /// Change display state while retaining geometry and verification; example: `sketches.set_visible(id, false)?`.
    pub fn set_visible(&mut self, id: DocumentSketchId, visible: bool) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.sketches[index].visible = visible;
        Ok(())
    }
    /// Show or hide every relation's mark; example: `sketches.set_constraints_shown(id, true)?`.
    pub fn set_constraints_shown(
        &mut self,
        id: DocumentSketchId,
        shown: bool,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.sketches[index].constraints_shown = shown;
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
pub(super) fn restore_constraints_shown(
    sketches: &mut DocumentSketches,
    recipe: &PlacedSketchRecipe,
    version: u32,
) -> Result<(), SketchError> {
    if recipe.constraints_shown.is_some() && version < 6 {
        return Err(SketchError::Definition(format!(
            "sketch {:?} constraints shown {:?} in recipe version {version}: expected version 6",
            recipe.id, recipe.constraints_shown
        )));
    }
    sketches.set_constraints_shown(recipe.id, recipe.constraints_shown.unwrap_or(false))
}
