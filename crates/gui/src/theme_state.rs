// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::theme::{ThemeLibrary, ThemePalette};

/// An immutable application theme snapshot; the GUI never edits its own library.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiThemeState {
    active: String,
    editable: bool,
    save_available: bool,
    names: Vec<String>,
    palette: ThemePalette,
    revision: u64,
}
impl GuiThemeState {
    /// Copy the library for a frame; example: `GuiThemeState::from_library(&library)`.
    pub fn from_library(library: &ThemeLibrary) -> Self {
        Self {
            active: library.active().name().into(),
            editable: library.active().kind() == &oblikovati_api::types::THEME_CUSTOM,
            save_available: false,
            names: library
                .themes()
                .iter()
                .map(|theme| theme.name().into())
                .collect(),
            palette: library.active().palette().clone(),
            revision: library.revision(),
        }
    }
    /// Attach the owner's storage capability; example: `state.with_custom_save_available(can_save)`.
    pub fn with_custom_save_available(mut self, available: bool) -> Self {
        self.save_available = available && self.editable;
        self
    }
    pub(crate) fn active_name(&self) -> &str {
        &self.active
    }
    pub(crate) fn editable(&self) -> bool {
        self.editable
    }
    pub(crate) fn save_available(&self) -> bool {
        self.save_available
    }
    /// Read semantic colours without mutable ownership; example: `state.palette()[&TOKEN_CHROME_TEXT]`.
    pub fn palette(&self) -> &ThemePalette {
        &self.palette
    }
    pub(crate) fn contains(&self, name: &str) -> bool {
        self.names.iter().any(|candidate| candidate == name)
    }
    pub(crate) fn controls(&self, ui: &imgui::Ui, actions: &mut Vec<crate::GuiShellAction>) {
        if let Some(name) =
            crate::appearance_choice::appearance_choice(ui, "Theme", &self.active, &self.names)
        {
            actions.push(crate::GuiShellAction::SetTheme(name));
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
