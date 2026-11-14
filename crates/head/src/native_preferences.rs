// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::theme::ThemeLibrary;
use oblikovati_persistence::{
    theme_file_preferences::ThemeFilePreferences,
    theme_files::{ThemeFileDecline, load_custom_themes},
    theme_preferences::{NativeThemePreferenceFiles, ThemePreferenceFile},
    user_config::user_config_directory,
};
use std::path::{Path, PathBuf};
type NativeThemePreferences = ThemeFilePreferences<NativeThemePreferenceFiles>;

pub(super) fn load_native_selection_filter(
    directory: &Path,
) -> Result<crate::selection_preferences::SelectionPreferenceFile<NativeThemePreferenceFiles>, String>
{
    crate::selection_preferences::SelectionPreferenceFile::load(
        &directory.join("selection-filter.json"),
        NativeThemePreferenceFiles,
    )
}

pub(super) fn load_native_marking_menu(
    directory: &Path,
) -> Result<
    oblikovati_persistence::marking_menu_preferences::MarkingMenuPreferenceFile<
        NativeThemePreferenceFiles,
    >,
    String,
> {
    oblikovati_persistence::marking_menu_preferences::MarkingMenuPreferenceFile::load(
        &directory.join("markingmenu.yaml"),
        NativeThemePreferenceFiles,
    )
    .map_err(|cause| cause.to_string())
}

pub(super) fn load_native_keymap(
    directory: &Path,
) -> Result<
    oblikovati_persistence::keymap_preferences::KeymapPreferenceFile<NativeThemePreferenceFiles>,
    String,
> {
    oblikovati_persistence::keymap_preferences::KeymapPreferenceFile::load(
        &directory.join("keymap.yaml"),
        NativeThemePreferenceFiles,
    )
    .map_err(|cause| cause.to_string())
}

pub(super) struct NativeThemeStartup {
    pub library: ThemeLibrary,
    pub preferences: NativeThemePreferences,
    pub declines: Vec<ThemeFileDecline>,
}
pub(super) fn load_native_themes(directory: &Path) -> Result<NativeThemeStartup, String> {
    let preferences = ThemePreferenceFile::load(
        &directory.join("preferences.yaml"),
        NativeThemePreferenceFiles,
    )
    .map_err(|cause| cause.to_string())?;
    let loaded = load_custom_themes(directory, &mut NativeThemePreferenceFiles)
        .map_err(|cause| cause.to_string())?;
    let library = ThemeLibrary::new(loaded.customs, preferences.active_name())?;
    let preferences =
        ThemeFilePreferences::new(preferences, NativeThemePreferenceFiles, loaded.sources)
            .with_custom_directory(directory.join("themes"));
    Ok(NativeThemeStartup {
        library,
        preferences,
        declines: loaded.declines,
    })
}

pub(super) fn native_config_directory() -> Result<PathBuf, String> {
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

/// Load options from the selected configuration directory.
/// Example: `load_native_application_options(config_dir)?`.
pub(super) fn load_native_application_options(
    directory: &Path,
) -> Result<
    oblikovati_persistence::application_options::ApplicationPreferenceFile<
        NativeThemePreferenceFiles,
    >,
    String,
> {
    let path = directory.join("options.yaml");
    oblikovati_persistence::application_options::ApplicationPreferenceFile::load(
        &path,
        NativeThemePreferenceFiles,
    )
    .map_err(|cause| cause.to_string())
}

/// The desktop's atomically replaced ribbon customization file.
pub(super) struct NativeRibbonSettings {
    path: PathBuf,
}
impl NativeRibbonSettings {
    pub(super) fn new(directory: &Path) -> Self {
        Self {
            path: directory.join("ribbon-customization.xml"),
        }
    }
}
impl oblikovati_gui::GuiRibbonSettingsStore for NativeRibbonSettings {
    fn load(&mut self) -> Result<Option<Vec<u8>>, String> {
        use std::io::Read;
        let file = match std::fs::File::open(&self.path) {
            Ok(file) => file,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(None),
            Err(error) => {
                return Err(format!(
                    "ribbon settings {:?}: expected a readable file: {error}",
                    self.path
                ));
            }
        };
        let mut bytes = Vec::new();
        file.take(1024 * 1024 + 1)
            .read_to_end(&mut bytes)
            .map_err(|error| {
                format!(
                    "ribbon settings {:?}: expected readable bytes: {error}",
                    self.path
                )
            })?;
        Ok(Some(bytes))
    }
    fn save(&mut self, bytes: &[u8]) -> Result<(), String> {
        oblikovati_persistence::atomic::replace_file(
            &self.path,
            bytes,
            &mut NativeThemePreferenceFiles,
        )
        .map_err(|error| error.to_string())
    }
}

pub(super) fn load_native_orientation_preferences(
    directory: &Path,
) -> Result<
    oblikovati_persistence::orientation_preferences::OrientationPreferenceFile<
        NativeThemePreferenceFiles,
    >,
    String,
> {
    oblikovati_persistence::orientation_preferences::OrientationPreferenceFile::load(
        &directory.join("ui-preferences.yaml"),
        NativeThemePreferenceFiles,
    )
    .map_err(|cause| cause.to_string())
}
