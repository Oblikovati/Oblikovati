// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    types::{GRADIENT_BACKGROUND, ONE_COLOR_BACKGROUND},
    wire::SetColorSchemeArgs,
};
use oblikovati_model::color_scheme::ColorSchemeRegistry;
use oblikovati_viewport::ViewportBackground;

/// A palette snapshot shares one owner with API selection and geometry-independent background.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiColorSchemeState {
    active: String,
    names: Vec<String>,
    background: ViewportBackground,
}
impl GuiColorSchemeState {
    /// Copy authored colours for a GUI frame; example: `GuiColorSchemeState::from_registry(&registry)?`.
    pub fn from_registry(registry: &ColorSchemeRegistry) -> Result<Self, String> {
        Ok(Self {
            active: registry.active().name.clone(),
            names: registry
                .schemes()
                .iter()
                .map(|scheme| scheme.name.clone())
                .collect(),
            background: registry_background(registry)?,
        })
    }
    /// Read the view's application colours; example: `viewport.set_background(state.background())?`.
    pub fn background(&self) -> ViewportBackground {
        self.background
    }
    pub(crate) fn contains(&self, name: &str) -> bool {
        self.names.iter().any(|candidate| candidate == name)
    }
    pub(crate) fn controls(&self, ui: &imgui::Ui, actions: &mut Vec<crate::GuiShellAction>) {
        ui.set_next_item_width(160.);
        let Some(_combo) = ui.begin_combo("Colour scheme", &self.active) else {
            return;
        };
        for name in &self.names {
            if ui
                .menu_item_config(name)
                .selected(name == &self.active)
                .build()
            {
                actions.push(crate::GuiShellAction::SetColorScheme(SetColorSchemeArgs {
                    name: name.clone(),
                }));
            }
        }
    }
}
fn registry_background(registry: &ColorSchemeRegistry) -> Result<ViewportBackground, String> {
    let scheme = registry.active();
    let rgb = |color: &oblikovati_api::types::Color| [color.r, color.g, color.b];
    match registry.background_type() {
        ONE_COLOR_BACKGROUND => Ok(ViewportBackground::Solid(rgb(&scheme.screen))),
        GRADIENT_BACKGROUND => Ok(ViewportBackground::VerticalGradient {
            top: rgb(&scheme.top),
            bottom: rgb(&scheme.bottom),
        }),
        other => Err(format!(
            "colour scheme {:?}, background type {}: expected a solid or gradient palette; image backgrounds require an image source",
            scheme.name, other.0
        )),
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn palette_snapshot_maps_authored_modes_and_refuses_missing_image_sources() {
        let mut registry = ColorSchemeRegistry::default();
        let state = GuiColorSchemeState::from_registry(&registry).unwrap();
        assert_eq!(
            state.background(),
            ViewportBackground::VerticalGradient {
                top: [60, 64, 72],
                bottom: [30, 32, 36]
            }
        );
        assert!(state.contains("High Contrast"));
        assert!(!state.contains("missing"));
        registry.set_active("High Contrast").unwrap();
        assert_eq!(
            GuiColorSchemeState::from_registry(&registry)
                .unwrap()
                .background(),
            ViewportBackground::Solid([0; 3])
        );
        registry
            .set_background_type(oblikovati_api::types::IMAGE_BACKGROUND)
            .unwrap();
        assert!(
            GuiColorSchemeState::from_registry(&registry)
                .unwrap_err()
                .contains("image source")
        );
    }
}
