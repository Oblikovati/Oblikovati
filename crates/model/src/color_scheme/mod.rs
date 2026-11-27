// SPDX-License-Identifier: GPL-2.0-only
//! Viewport palettes have an application owner separate from semantic UI themes.
use oblikovati_api::types::{
    BackgroundTypeEnum, Color, GRADIENT_BACKGROUND, IMAGE_BACKGROUND, ONE_COLOR_BACKGROUND,
};
mod builtins;

#[derive(Clone, Debug, PartialEq)]
pub struct ColorScheme {
    pub name: String,
    pub background_type: BackgroundTypeEnum,
    pub screen: Color,
    pub top: Color,
    pub bottom: Color,
    pub highlight: Color,
    pub primary_select: Color,
    pub secondary_select: Color,
}
#[derive(Clone, Debug, PartialEq)]
pub struct ColorSchemeRegistry {
    schemes: Vec<ColorScheme>,
    active: usize,
    background: BackgroundTypeEnum,
}
impl Default for ColorSchemeRegistry {
    fn default() -> Self {
        let schemes = builtins::schemes();
        Self {
            background: schemes[0].background_type,
            schemes,
            active: 0,
        }
    }
}
impl ColorSchemeRegistry {
    /// Read the gallery in Go picker order; example: `registry.schemes()[0].name`.
    pub fn schemes(&self) -> &[ColorScheme] {
        &self.schemes
    }
    /// Read the selected viewport palette; example: `registry.active().screen`.
    pub fn active(&self) -> &ColorScheme {
        &self.schemes[self.active]
    }
    /// Read the application background override; example: `registry.background_type()`.
    pub fn background_type(&self) -> BackgroundTypeEnum {
        self.background
    }
    /// Select a scheme and restore its background mode; example: `registry.set_active("Sky")?`.
    pub fn set_active(&mut self, name: &str) -> Result<(), String> {
        let index = self
            .schemes
            .iter()
            .position(|scheme| scheme.name == name)
            .ok_or_else(|| {
                format!(
                    "color scheme {name:?}: expected one of {:?}",
                    self.schemes
                        .iter()
                        .map(|scheme| &scheme.name)
                        .collect::<Vec<_>>()
                )
            })?;
        self.active = index;
        self.background = self.schemes[index].background_type;
        Ok(())
    }
    /// Override the application background mode; example: `registry.set_background_type(ONE_COLOR_BACKGROUND)?`.
    pub fn set_background_type(&mut self, background: BackgroundTypeEnum) -> Result<(), String> {
        if ![ONE_COLOR_BACKGROUND, GRADIENT_BACKGROUND, IMAGE_BACKGROUND].contains(&background) {
            return Err(format!(
                "background type {}: expected a defined BackgroundTypeEnum",
                background.0
            ));
        }
        self.background = background;
        Ok(())
    }
}
impl oblikovati_api::contract::ColorScheme for ColorScheme {
    fn name(&self) -> String {
        self.name.clone()
    }
    fn screen_color(&self) -> Color {
        self.screen.clone()
    }
    fn top_screen_color(&self) -> Color {
        self.top.clone()
    }
    fn bottom_screen_color(&self) -> Color {
        self.bottom.clone()
    }
    fn highlight_color(&self) -> Color {
        self.highlight.clone()
    }
    fn primary_select_color(&self) -> Color {
        self.primary_select.clone()
    }
    fn secondary_select_color(&self) -> Color {
        self.secondary_select.clone()
    }
}
const _: fn(&ColorScheme) -> &dyn oblikovati_api::contract::ColorScheme = |scheme| scheme;
