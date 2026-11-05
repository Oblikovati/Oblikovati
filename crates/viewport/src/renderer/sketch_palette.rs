// SPDX-License-Identifier: GPL-2.0-only
use super::{
    RenderView, sketch_display::SketchDisplayOverlay, sketch_preview::SketchPreviewOverlay,
};
use crate::ViewportSketchPalette;
impl RenderView {
    pub(crate) fn set_sketch_palette(
        &mut self,
        palette: &ViewportSketchPalette,
    ) -> Result<bool, String> {
        if self.sketch_palette == *palette {
            return Ok(false);
        }
        let deflection = self.sketch_deflection()?;
        let sketches = self.restyle_committed(palette, deflection, &self.sketch_selection)?;
        let preview = self.restyle_preview(palette, deflection)?;
        // Build both replacements before publishing either, so a declined preview keeps both old overlays.
        sketches.show();
        preview.show();
        self.sketches = sketches;
        self.preview = preview;
        self.sketch_palette = palette.clone();
        Ok(true)
    }
    fn restyle_committed(
        &self,
        palette: &ViewportSketchPalette,
        deflection: f64,
        selected: &crate::ViewportSketchSelection,
    ) -> Result<SketchDisplayOverlay, String> {
        let source = self
            .sketches
            .source
            .as_ref()
            .map_or(&[][..], |(source, _)| source.as_slice());
        SketchDisplayOverlay::replacement(&self.viewer, source, deflection, palette, selected)
    }
    fn restyle_preview(
        &self,
        palette: &ViewportSketchPalette,
        deflection: f64,
    ) -> Result<SketchPreviewOverlay, String> {
        SketchPreviewOverlay::replacement(
            &self.viewer,
            self.preview_source().as_ref(),
            deflection,
            &palette.preview,
        )
    }
}

impl RenderView {
    pub(crate) fn set_sketch_selection(
        &mut self,
        selected: &crate::ViewportSketchSelection,
    ) -> Result<bool, String> {
        let selected = selected.normalized();
        if self.sketch_selection == selected {
            return Ok(false);
        }
        let sketches =
            self.restyle_committed(&self.sketch_palette, self.sketch_deflection()?, &selected)?;
        sketches.show();
        self.sketches = sketches;
        self.sketch_selection = selected;
        Ok(true)
    }
}
