// SPDX-License-Identifier: GPL-2.0-only
use super::{SketchError, SketchPathMember, SketchPlacementGeometry, SketchPlacementRecipe};
impl SketchPlacementRecipe {
    /// Apply armed curve roles while preserving a tool's intrinsic construction members.
    /// Example: `recipe.apply_creation_formats(true, false)?;` before preview and commit.
    pub fn apply_creation_formats(
        &mut self,
        construction: bool,
        centerline: bool,
    ) -> Result<(), SketchError> {
        self.require_curve_mode_members(construction)?;
        for member in &mut self.members {
            member.construction |= construction;
            member.centerline |= centerline
                && matches!(
                    member.geometry,
                    SketchPlacementGeometry::Path(SketchPathMember::Line(_))
                );
        }
        Ok(())
    }
    fn require_curve_mode_members(&self, construction: bool) -> Result<(), SketchError> {
        if construction
            && self
                .members
                .iter()
                .any(|member| matches!(member.geometry, SketchPlacementGeometry::Point(_)))
        {
            return Err(SketchError::Definition(format!(
                "construction placement members {:?}: expected curves; construction points are unsupported",
                self.members
            )));
        }
        Ok(())
    }
}
