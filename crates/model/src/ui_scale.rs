// SPDX-License-Identifier: GPL-2.0-only
//! UI content preferences are separate from native backing-pixel scale (#3850).

/// Validated user scale factors; document edits and undo do not change them.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct UiScaleOptions {
    font: f32,
    icon: f32,
}
impl Default for UiScaleOptions {
    fn default() -> Self {
        Self { font: 1., icon: 1. }
    }
}
impl UiScaleOptions {
    /// Clamp finite edits to Go's bounds; example: `UiScaleOptions::new(1.5, 2.)?`.
    pub fn new(font: f32, icon: f32) -> Result<Self, String> {
        if !font.is_finite() || !icon.is_finite() {
            return Err(format!(
                "UI scale font={font}, icon={icon}: expected finite scale factors"
            ));
        }
        Ok(Self {
            font: font.clamp(0.5, 2.),
            icon: icon.clamp(0.5, 2.5),
        })
    }
    /// Keep Go's missing/non-positive stored-value default; example: `UiScaleOptions::from_stored(0., 1.5)?`.
    pub fn from_stored(font: f32, icon: f32) -> Result<Self, String> {
        Self::new(
            if font.is_finite() && font <= 0. {
                1.
            } else {
                font
            },
            if icon.is_finite() && icon <= 0. {
                1.
            } else {
                icon
            },
        )
    }
    /// Read text size independently of framebuffer density; example: `options.font()`.
    pub fn font(self) -> f32 {
        self.font
    }
    /// Read vector-icon size independently of text size; example: `options.icon()`.
    pub fn icon(self) -> f32 {
        self.icon
    }
}
/// An injected application preference owner publishes values only after successful storage.
pub trait UiScalePreferenceAccess {
    /// Read validated scale factors; example: `preferences.ui_scale().font()`.
    fn ui_scale(&self) -> UiScaleOptions;
    /// Persist an edit independently of document history; example: `preferences.set_ui_scale(options)?`.
    fn set_ui_scale(&mut self, options: UiScaleOptions) -> Result<(), String>;
}

#[cfg(test)]
#[path = "../tests/ui_scale/model.rs"]
mod tests;
