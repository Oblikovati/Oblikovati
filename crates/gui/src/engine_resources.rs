// SPDX-License-Identifier: GPL-2.0-only
//! Owner font resources and logical-to-physical scale.
use super::*;
impl GuiFrameRequest {
    /// Convert UI layout units to backing pixels; native backing density stays independently recorded.
    /// Example: backing 1 and content 2 produce `layout_scale() == [2.; 2]`.
    pub(crate) fn layout_scale(&self) -> [f32; 2] {
        let content = self.commands.ui_scale.unwrap_or_default().font();
        self.framebuffer_scale.map(|backing| backing * content)
    }
}
impl GuiFrameEngine {
    /// Report whether the tab preview can use the next frame.
    /// Example: `engine.needs_frame_pixels()` before reading the composed surface.
    pub fn needs_frame_pixels(&self) -> bool {
        self.document_tabs.previews.needs_frame_pixels()
    }

    /// Report whether the displayed region needs a new sample of the composed frame.
    /// Example: `engine.needs_frame_pixels_at(view.completed_generations())`.
    pub fn needs_frame_pixels_at(&self, completed: Option<[u64; 4]>) -> bool {
        self.document_tabs.previews.needs_frame_pixels_at(completed)
    }

    /// Publish preview pixels with their native frame identity.
    /// Example: `engine.note_presented_window_capture(&capture)` after presenting.
    pub fn note_presented_window_capture(
        &mut self,
        capture: &oblikovati_viewport::ViewportCapture,
    ) {
        let generations = [
            capture.scene_generation,
            capture.camera_generation,
            capture.surface_generation,
            capture.overlay_generation,
        ];
        self.document_tabs
            .previews
            .note_frame_at(capture.extent, &capture.rgba, Some(generations));
    }

    /// Publish borrowed preview pixels from an accepted viewport frame.
    /// Example: `engine.note_presented_window_preview(extent, generations, pixels)`.
    pub fn note_presented_window_preview(
        &mut self,
        extent: [u32; 2],
        generations: [u64; 4],
        pixels: oblikovati_viewport::ViewportPreviewPixels<'_>,
    ) {
        self.document_tabs
            .previews
            .note_pixels_at(extent, pixels, Some(generations));
    }

    pub(super) fn refresh_chrome(&mut self, request: &GuiFrameRequest) -> bool {
        // Docked tab bars use their windows' colors from the previous frame.
        request.commands.theme.as_ref().is_some_and(|theme| {
            crate::theme::apply_chrome_theme(&mut self.context.style_mut().colors, theme.palette())
        })
    }

    pub(super) fn refresh_font_scale(&mut self, request: &GuiFrameRequest) -> Result<(), String> {
        let scale = crate::ui_scale::font_raster_scale(request)?;
        if self.raster_scale == scale {
            return Ok(());
        }
        self.atlas = crate::ui_scale::rebuild_font_atlas(&mut self.context, scale);
        self.resources = self.resources.with_font(self.atlas.clone())?;
        self.raster_scale = scale;
        self.ribbon.panels = Default::default();
        Ok(())
    }
}

#[cfg(test)]
#[path = "../tests/ui_scale/engine.rs"]
mod scale_tests;

#[cfg(test)]
#[path = "../tests/textures/owner.rs"]
mod texture_tests;

#[cfg(test)]
#[path = "../tests/icons/engine.rs"]
mod tests;

#[cfg(test)]
#[path = "../tests/application/ribbon-display/panel_selector.rs"]
mod panel_selector_tests;

#[cfg(test)]
#[path = "../tests/browser/click.rs"]
mod browser_click_tests;

#[cfg(test)]
#[path = "../tests/document-tabs/click.rs"]
mod document_tab_click_tests;

#[cfg(test)]
#[path = "../tests/icons/shell_pointer.rs"]
mod shell_pointer_tests;
