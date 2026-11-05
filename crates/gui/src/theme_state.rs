// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::theme::{ThemeLibrary, ThemePalette};

/// An immutable application theme snapshot; the GUI never edits its own library.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThemeState {
    active: String,
    names: Vec<String>,
    palette: ThemePalette,
    revision: u64,
}
impl GuiThemeState {
    /// Copy the library for a frame; example: `GuiThemeState::from_library(&library)`.
    pub fn from_library(library: &ThemeLibrary) -> Self {
        Self {
            active: library.active().name().into(),
            names: library
                .themes()
                .iter()
                .map(|theme| theme.name().into())
                .collect(),
            palette: library.active().palette().clone(),
            revision: library.revision(),
        }
    }
    /// Read semantic colours without mutable ownership; example: `state.palette()[&TOKEN_CHROME_TEXT]`.
    pub fn palette(&self) -> &ThemePalette {
        &self.palette
    }
    pub(crate) fn contains(&self, name: &str) -> bool {
        self.names.iter().any(|candidate| candidate == name)
    }
    pub(crate) fn controls(&self, ui: &imgui::Ui, actions: &mut Vec<crate::GuiShellAction>) {
        ui.set_next_item_width(160.);
        let Some(_combo) = ui.begin_combo("Theme", &self.active) else {
            return;
        };
        for name in &self.names {
            if ui
                .menu_item_config(name)
                .selected(name == &self.active)
                .build()
            {
                actions.push(crate::GuiShellAction::SetTheme(name.clone()));
            }
        }
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn copied_theme_states_change_on_selection_without_editing_the_source() {
        let mut library = ThemeLibrary::new(Vec::new(), "Dark").unwrap();
        let before = GuiThemeState::from_library(&library);
        assert!(before.contains("Light"));
        assert!(!before.contains("missing"));
        library.set_active("Light");
        let after = GuiThemeState::from_library(&library);
        assert_ne!(before, after);
        assert_eq!(before.active, "Dark");
        assert_eq!(after.active, "Light");
        assert_eq!(after.palette(), library.active().palette());
    }
}
