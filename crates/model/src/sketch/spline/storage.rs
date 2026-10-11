// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch::{Sketch2D, SketchError, SketchRecipe};
impl Sketch2D {
    pub(in crate::sketch) fn capture_spline_schema(&self, recipe: &mut SketchRecipe) {
        if self.splines.is_empty() && self.offset_splines.is_empty() {
            return;
        }
        recipe.version = 10;
        self.capture_complete_schema_metadata(recipe);
        recipe.splines = Some(self.splines.clone());
        recipe.offset_splines = Some(self.offset_splines.clone());
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
        (10, Some(parents), Some(offsets)) => Ok(Some((parents, offsets))),
        shape => Err(SketchError::Definition(format!(
            "spline recipe {shape:?}: expected no spline fields in versions 1-9 or explicit parent/offset lists in version 10"
        ))),
    }
}
