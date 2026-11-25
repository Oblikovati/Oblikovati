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
