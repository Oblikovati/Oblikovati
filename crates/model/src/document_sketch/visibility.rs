// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSketchId, DocumentSketches, PlacedSketch, PlacedSketchRecipe};
use crate::sketch::SketchError;

impl PlacedSketch {
    /// Read individually hidden geometric glyph IDs; example: `sketch.hidden_constraint_glyphs()`.
    pub fn hidden_constraint_glyphs(
        &self,
    ) -> &std::collections::BTreeSet<crate::sketch::SketchConstraintId> {
        &self.hidden_constraint_glyphs
    }
    /// Read saved display state without requiring a successful solve; example: `sketch.visible()`.
    pub fn visible(&self) -> bool {
        self.visible
    }
    /// How its lines are drawn; example: `sketch.line_style().color`.
    pub fn line_style(&self) -> &super::SketchLineStyle {
        &self.line_style
    }
    /// Whether relations draw their marks; example: `sketch.constraints_shown()`.
    pub fn constraints_shown(&self) -> bool {
        self.constraints_shown
    }
}
impl DocumentSketches {
    /// Change one geometric glyph without changing its relation.
    /// Example: `sketches.set_constraint_glyph_visible(sketch, relation, false)?`.
    pub fn set_constraint_glyph_visible(
        &mut self,
        id: DocumentSketchId,
        relation: crate::sketch::SketchConstraintId,
        visible: bool,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        if !self.sketches[index]
            .geometry
            .has_geometric_relation(relation)
        {
            return Err(SketchError::Definition(format!(
                "sketch constraint {relation:?}: expected a geometric relation for glyph visibility"
            )));
        }
        if visible {
            self.sketches[index]
                .hidden_constraint_glyphs
                .remove(&relation);
        } else {
            self.sketches[index]
                .hidden_constraint_glyphs
                .insert(relation);
        }
        Ok(())
    }
    /// Change display state while retaining geometry and verification; example: `sketches.set_visible(id, false)?`.
    pub fn set_visible(&mut self, id: DocumentSketchId, visible: bool) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.sketches[index].visible = visible;
        Ok(())
    }
    /// Change how its lines are drawn; example: `sketches.set_line_style(id, style)?`.
    pub fn set_line_style(
        &mut self,
        id: DocumentSketchId,
        style: super::SketchLineStyle,
    ) -> Result<(), SketchError> {
        let index = self.index(id)?;
        self.sketches[index].line_style = style;
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
        if shown {
            self.sketches[index].hidden_constraint_glyphs.clear();
        }
        Ok(())
    }
}

/// Restore visibility after its geometric relation identities exist.
pub(super) fn restore_hidden_glyphs(
    sketches: &mut DocumentSketches,
    recipe: &PlacedSketchRecipe,
    version: u32,
) -> Result<(), SketchError> {
    if !recipe.hidden_constraint_glyphs.is_empty() && version < 17 {
        return Err(SketchError::Definition(format!(
            "hidden constraint glyphs in recipe version {version}: expected version 17 or later"
        )));
    }
    let mut unique = std::collections::BTreeSet::new();
    for relation in &recipe.hidden_constraint_glyphs {
        if !unique.insert(*relation) {
            return Err(SketchError::Definition(format!(
                "duplicate hidden constraint glyph {relation:?}: expected unique geometric relation identities"
            )));
        }
        sketches.set_constraint_glyph_visible(recipe.id, *relation, false)?;
    }
    Ok(())
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
pub(super) fn restore_line_style(
    sketches: &mut DocumentSketches,
    recipe: &PlacedSketchRecipe,
    version: u32,
) -> Result<(), SketchError> {
    if recipe.line_style.is_some() && version < 7 {
        return Err(SketchError::Definition(format!(
            "sketch {:?} line style {:?} in recipe version {version}: expected version 7",
            recipe.id, recipe.line_style
        )));
    }
    sketches.set_line_style(recipe.id, recipe.line_style.clone().unwrap_or_default())
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

impl oblikovati_api::contract::ConstraintGlyphVisibility for PlacedSketch {
    fn constraint_glyphs_shown(&self) -> bool {
        self.constraints_shown
    }
    fn constraint_glyph_visible(&self, id: u64) -> Result<bool, oblikovati_api::Error> {
        let relation = crate::sketch::SketchConstraintId(id);
        if !self.geometry.has_geometric_relation(relation) {
            return Err(oblikovati_api::Error(format!(
                "sketch constraint {id}: expected a geometric relation for glyph visibility"
            )));
        }
        Ok(self.constraints_shown && !self.hidden_constraint_glyphs.contains(&relation))
    }
}
