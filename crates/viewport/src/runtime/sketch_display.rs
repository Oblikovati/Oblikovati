// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl Viewport {
    /// Replace committed sketch presentation without modifying solid meshes or transient previews.
    /// Example: `viewport.set_sketch_display(&sketches)?` after a document history change.
    pub fn set_sketch_display(
        &mut self,
        sketches: &[crate::ViewportSketchDisplay],
    ) -> Result<bool, String> {
        self.update_overlay(|render| render.sketch_display(sketches))
    }
    /// Inspect every local display failure; example: `viewport.sketch_display_declines()?` for visible feedback.
    pub fn sketch_display_declines(&self) -> Result<&[crate::ViewportSketchDecline], String> {
        Ok(self.live()?.sketch_declines())
    }
    pub(super) fn refresh_sketch_display(&mut self) -> Result<(), String> {
        self.update_overlay(RenderView::refresh_sketch_display)
            .map(|_| ())
    }
}
