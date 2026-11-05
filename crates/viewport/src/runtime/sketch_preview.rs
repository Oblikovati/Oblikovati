// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl Viewport {
    /// Replace or clear the transient authoring outline; example: `viewport.set_sketch_preview(None)?` after commit.
    pub fn set_sketch_preview(
        &mut self,
        preview: Option<&crate::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        self.update_overlay(|render| render.sketch_preview(preview))
    }
}
