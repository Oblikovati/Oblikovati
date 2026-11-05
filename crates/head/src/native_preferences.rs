// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::theme::ThemeLibrary;
use oblikovati_persistence::{
    theme_files::{ThemeFileDecline, load_custom_themes},
    theme_preferences::{NativeThemePreferenceFiles, ThemePreferenceFile},
    user_config::user_config_directory,
};
use std::path::{Path, PathBuf};
type NativeThemePreferences = ThemePreferenceFile<NativeThemePreferenceFiles>;

pub(super) struct NativeThemeStartup {
    pub library: ThemeLibrary,
    pub preferences: NativeThemePreferences,
    pub declines: Vec<ThemeFileDecline>,
}
pub(super) fn load_native_themes() -> Result<NativeThemeStartup, String> {
    let directory = native_config_directory()?;
    let preferences = ThemePreferenceFile::load(
        &directory.join("preferences.yaml"),
        NativeThemePreferenceFiles,
    )
    .map_err(|cause| cause.to_string())?;
    let loaded = load_custom_themes(&directory, &mut NativeThemePreferenceFiles)
        .map_err(|cause| cause.to_string())?;
    let library = ThemeLibrary::new(loaded.customs, preferences.active_name())?;
    Ok(NativeThemeStartup {
        library,
        preferences,
        declines: loaded.declines,
    })
}

fn native_config_directory() -> Result<PathBuf, String> {
    let explicit = std::env::var_os("OBK_CONFIG_DIR");
    let home = std::env::var_os("HOME");
    let roaming = std::env::var_os("APPDATA");
    user_config_directory(
        explicit.as_deref().map(Path::new),
        home.as_deref().map(Path::new),
        roaming.as_deref().map(Path::new),
        cfg!(windows),
    )
}
