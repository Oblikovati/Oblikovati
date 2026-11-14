// SPDX-License-Identifier: GPL-2.0-only
//! Default theme and palette services for the desktop frame owner.
macro_rules! theme_work_methods {
    () => {
        /// Apply a live custom colour preview; example: `work.edit_theme_color_from_gui(name, token, value)?`.
        fn edit_theme_color_from_gui(
            &mut self,
            name: &str,
            token: &oblikovati_api::types::ThemeToken,
            _value: oblikovati_api::types::Rgba,
        ) -> Result<bool, String> {
            Err(format!(
                "theme {name:?}, token {token:?}: expected an editable application theme owner"
            ))
        }
        /// Copy the active theme; example: `work.duplicate_theme_from_gui("Dark", "Workshop")?`.
        fn duplicate_theme_from_gui(&mut self, base: &str, name: &str) -> Result<bool, String> {
            Err(format!(
                "theme {base:?} -> {name:?}: expected an application theme owner"
            ))
        }
        /// Delete an active custom; example: `work.delete_theme_from_gui("Workshop")?`.
        fn delete_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
            Err(format!(
                "theme {name:?}: expected an application theme owner"
            ))
        }
        /// Save the active custom snapshot; example: `work.save_theme_from_gui(name)?`.
        fn save_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
            Err(format!(
                "theme {name:?}: expected an application theme storage owner"
            ))
        }
        /// Apply the canonical palette selection; example: `work.select_color_scheme_from_gui(&request)?`.
        fn select_color_scheme_from_gui(
            &mut self,
            request: &oblikovati_api::wire::SetColorSchemeArgs,
        ) -> Result<bool, String> {
            Err(format!(
                "colour scheme {:?}: expected an application palette owner",
                request.name
            ))
        }
        /// Select the application theme; example: `work.select_theme_from_gui("Light")?`.
        fn select_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
            Err(format!(
                "theme {name:?}: expected an application theme owner"
            ))
        }
    };
}
pub(crate) use theme_work_methods;
