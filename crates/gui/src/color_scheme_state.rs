// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{
    types::{GRADIENT_BACKGROUND, ONE_COLOR_BACKGROUND, TOKEN_VIEWPORT_BG},
    wire::SetColorSchemeArgs,
};
use oblikovati_model::{color_scheme::ColorSchemeRegistry, theme::ThemePalette};
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
        Self::from_registry_and_theme(registry, None)
    }
    /// Follow the theme in the default scheme while preserving named and background mode overrides.
    /// Example: `GuiColorSchemeState::from_registry_and_theme(&registry, Some(theme.palette()))`.
    pub fn from_registry_and_theme(
        registry: &ColorSchemeRegistry,
        theme: Option<&ThemePalette>,
    ) -> Result<Self, String> {
        Ok(Self {
            active: registry.active().name.clone(),
            names: registry
                .schemes()
                .iter()
                .map(|scheme| scheme.name.clone())
                .collect(),
            background: studio_background(registry, theme)
                .map_or_else(|| registry_background(registry), Ok)?,
        })
    }
    /// Read the view's application colours; example: `viewport.set_background(state.background())?`.
    pub fn background(&self) -> ViewportBackground {
        self.background
    }
    pub(crate) fn contrasting_ink(&self, preferred: [f32; 4]) -> [f32; 4] {
        let samples = match self.background {
            ViewportBackground::Solid(rgb) => [rgb; 2],
            ViewportBackground::VerticalGradient { top, bottom } => [top, bottom],
        };
        let surfaces = samples.map(|rgb| luminance(rgb.map(|value| f32::from(value) / 255.)));
        let contrast = |[r, g, b, _]: [f32; 4]| minimum_contrast([r, g, b], surfaces);
        if contrast(preferred) >= 4.5 {
            return preferred;
        }
        let dark = [0.067, 0.075, 0.094, preferred[3]];
        let light = [0.925, 0.933, 0.949, preferred[3]];
        if contrast(dark) > contrast(light) {
            dark
        } else {
            light
        }
    }
    pub(crate) fn contains(&self, name: &str) -> bool {
        self.names.iter().any(|candidate| candidate == name)
    }
    pub(crate) fn controls(&self, ui: &imgui::Ui, actions: &mut Vec<crate::GuiShellAction>) {
        if let Some(name) = crate::appearance_choice::appearance_choice(
            ui,
            "Colour scheme",
            &self.active,
            &self.names,
        ) {
            actions.push(crate::GuiShellAction::SetColorScheme(SetColorSchemeArgs {
                name,
            }));
        }
    }
}
fn studio_background(
    registry: &ColorSchemeRegistry,
    theme: Option<&ThemePalette>,
) -> Option<ViewportBackground> {
    if registry.active().name != "Default" || registry.background_type() != GRADIENT_BACKGROUND {
        return None;
    }
    let base = &theme?[&TOKEN_VIEWPORT_BG];
    let top = [base.r, base.g, base.b].map(|value| (value * 255.).round() as u8);
    Some(match top {
        [0, 0, 0] => ViewportBackground::Solid(top),
        _ if luminance([base.r, base.g, base.b]) > 0.5 => ViewportBackground::VerticalGradient {
            top,
            bottom: [228, 232, 240],
        },
        _ => ViewportBackground::VerticalGradient {
            top,
            bottom: [30, 33, 42],
        },
    })
}
fn luminance(rgb: [f32; 3]) -> f32 {
    let linear = rgb.map(|value| {
        if value <= 0.04045 {
            value / 12.92
        } else {
            ((value + 0.055) / 1.055).powf(2.4)
        }
    });
    linear[0] * 0.2126 + linear[1] * 0.7152 + linear[2] * 0.0722
}
fn minimum_contrast(rgb: [f32; 3], backdrops: [f32; 2]) -> f32 {
    let ink = luminance(rgb);
    let ratio = |surface: f32| (ink.max(surface) + 0.05) / (ink.min(surface) + 0.05);
    ratio(backdrops[0]).min(ratio(backdrops[1]))
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
                top: [20, 22, 27],
                bottom: [30, 33, 42]
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

    #[test]
    fn studio_default_follows_the_theme_and_named_schemes_override_it() {
        let mut registry = ColorSchemeRegistry::default();
        let light = oblikovati_model::theme::ThemeDefinition::light().unwrap();
        let snapshot = |registry: &ColorSchemeRegistry| {
            GuiColorSchemeState::from_registry_and_theme(registry, Some(light.palette())).unwrap()
        };
        let ink = [0.067, 0.075, 0.094, 1.];
        let studio = snapshot(&registry);
        assert_eq!(
            studio.background(),
            ViewportBackground::VerticalGradient {
                top: [244, 246, 249],
                bottom: [228, 232, 240],
            }
        );
        assert_eq!(studio.contrasting_ink(ink), ink);
        registry.set_active("Sky").unwrap();
        assert_eq!(
            snapshot(&registry).background(),
            ViewportBackground::VerticalGradient {
                top: [150, 190, 235],
                bottom: [225, 235, 245],
            }
        );
        registry.set_active("High Contrast").unwrap();
        let override_state = snapshot(&registry);
        assert_eq!(
            override_state.background(),
            ViewportBackground::Solid([0; 3])
        );
        assert!(override_state.contrasting_ink(ink)[0] > 0.9);
    }
}
