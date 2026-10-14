// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch::{Sketch2D, SketchError, SketchRecipe};
impl Sketch2D {
    pub(in crate::sketch) fn capture_spline_schema(&self, recipe: &mut SketchRecipe) {
        if self.splines.is_empty() && self.offset_splines.is_empty() {
            return;
        }
        recipe.version = if !self.spline_handles.is_empty() {
            13
        } else if super::dimension::has_offset_dimension(recipe) {
            11
        } else {
            10
        };
        self.capture_complete_schema_metadata(recipe);
        recipe.splines = Some(self.splines.clone());
        recipe.offset_splines = Some(self.offset_splines.clone());
        recipe.spline_handles =
            (!self.spline_handles.is_empty()).then(|| self.spline_handles.clone());
    }
    pub(in crate::sketch) fn restore_splines(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        let Some((parents, offsets)) = spline_records(recipe)? else {
            return Ok(());
        };
        for parent in parents {
            self.add_spline(parent.clone())?;
        }
        for offset in offsets {
            self.add_offset_spline(*offset)?;
        }
        for handle in recipe.spline_handles.as_deref().unwrap_or_default() {
            self.add_spline_handle(*handle)?;
        }
        Ok(())
    }
}
type SplineRecords<'a> = (
    &'a [super::SketchSpline2D],
    &'a [super::SketchOffsetSpline2D],
);
pub(super) fn spline_records(
    recipe: &SketchRecipe,
) -> Result<Option<SplineRecords<'_>>, SketchError> {
    match (recipe.version, &recipe.splines, &recipe.offset_splines) {
        (1..=9, None, None) => Ok(None),
        (10..=13, Some(parents), Some(offsets)) if recipe.spline_handles.is_none() => {
            Ok(Some((parents, offsets)))
        }
        (13, Some(parents), Some(offsets)) if recipe.spline_handles.is_some() => {
            Ok(Some((parents, offsets)))
        }
        shape => Err(SketchError::Definition(format!(
            "spline recipe {shape:?}: expected explicit parent/offset lists, and handles only in version 13"
        ))),
    }
}
