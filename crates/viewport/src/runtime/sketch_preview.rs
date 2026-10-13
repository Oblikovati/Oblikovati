// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl Viewport {
    /// Replace or clear the transient authoring outline; example: `viewport.set_sketch_preview(None)?` after commit.
    pub fn set_sketch_preview(
        &mut self,
        preview: Option<&crate::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        self.live()?;
        self.refresh_extent()?;
        let next = next_generation(self.overlay)?;
        if !self
            .render
            .as_mut()
            .expect("live view")
            .sketch_preview(preview)?
        {
            return Ok(false);
        }
        self.overlay = next;
        self.completed = None;
        Ok(true)
    }
}
