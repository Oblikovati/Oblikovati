// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    atomic::{AtomicFiles, replace_file},
    load::PackageFileReader,
};
use oblikovati_model::theme::ThemeDefinition;
use std::{
    collections::BTreeSet,
    ffi::OsString,
    io,
    path::{Path, PathBuf},
};

/// Directory discovery is injected separately from whole-file reads and atomic writes.
pub trait ThemeDirectoryFiles: PackageFileReader {
    /// Return base names; an absent directory is empty. Example: `files.theme_entries(directory)?`.
    fn theme_entries(&mut self, directory: &Path) -> io::Result<Vec<OsString>>;
}
#[derive(Debug)]
pub struct ThemeFileDecline {
    pub path: PathBuf,
    pub cause: String,
}
#[derive(Debug)]
pub struct LoadedCustomThemes {
    pub customs: Vec<ThemeDefinition>,
    pub declines: Vec<ThemeFileDecline>,
}
/// Load valid custom XML in filename order, retaining per-file declines; example: `load_custom_themes(root, &mut files)?`.
pub fn load_custom_themes<Files: ThemeDirectoryFiles>(
    root: &Path,
    files: &mut Files,
) -> io::Result<LoadedCustomThemes> {
    let directory = root.join("themes");
    let entries = discover_theme_entries(&directory, files)?;
    let mut loaded = LoadedCustomThemes {
        customs: Vec::new(),
        declines: Vec::new(),
    };
    let mut names: BTreeSet<String> = ["Dark".into(), "Light".into()].into();
    for name in entries {
        load_entry(&directory, &name, files, &mut names, &mut loaded);
    }
    Ok(loaded)
}
fn load_entry<Files: ThemeDirectoryFiles>(
    directory: &Path,
    name: &OsString,
    files: &mut Files,
    names: &mut BTreeSet<String>,
    loaded: &mut LoadedCustomThemes,
) {
    if !name.to_string_lossy().ends_with(".xml") {
        return;
    }
    let path = directory.join(name);
    match read_custom(&path, name, files, names) {
        Ok(theme) => loaded.customs.push(theme),
        Err(cause) => loaded.declines.push(ThemeFileDecline { path, cause }),
    }
}
fn read_custom<Files: PackageFileReader>(
    path: &Path,
    name: &OsString,
    files: &mut Files,
    names: &mut BTreeSet<String>,
) -> Result<ThemeDefinition, String> {
    let fallback = custom_filename_stem(name)?;
    let bytes = files
        .read(path)
        .map_err(|cause| format!("theme file {path:?}: expected readable XML: {cause}"))?;
    let theme = ThemeDefinition::from_xml(&bytes, fallback, oblikovati_api::types::THEME_CUSTOM)?;
    if !names.insert(theme.name().into()) {
        return Err(format!(
            "theme name {:?}: expected a unique custom name distinct from Dark and Light",
            theme.name()
        ));
    }
    Ok(theme)
}
/// Export one complete custom snapshot atomically; example: `save_custom_theme(path, &theme, &mut files)?`.
pub fn save_custom_theme<Files: AtomicFiles>(
    path: &Path,
    theme: &ThemeDefinition,
    files: &mut Files,
) -> Result<(), String> {
    if theme.kind() != &oblikovati_api::types::THEME_CUSTOM {
        return Err(format!(
            "theme {:?} ({:?}): expected an exportable custom theme",
            theme.name(),
            theme.kind()
        ));
    }
    let encoded = theme.to_xml()?;
    replace_file(path, &encoded, files).map_err(|cause| cause.to_string())
}

fn discover_theme_entries<Files: ThemeDirectoryFiles>(
    directory: &Path,
    files: &mut Files,
) -> io::Result<Vec<OsString>> {
    let mut entries = files.theme_entries(directory).map_err(|cause| {
        io::Error::new(
            cause.kind(),
            format!("theme directory {directory:?}: expected readable entries: {cause}"),
        )
    })?;
    entries.sort_by_cached_key(|name| name.to_string_lossy().into_owned());
    Ok(entries)
}
fn custom_filename_stem(name: &OsString) -> Result<&str, String> {
    if Path::new(name).file_name() != Some(name.as_os_str()) {
        return Err(format!(
            "theme filename {name:?}: expected a directory base name"
        ));
    }
    name.to_str()
        .and_then(|name| name.strip_suffix(".xml"))
        .ok_or_else(|| format!("theme filename {name:?}: expected a UTF-8 XML file stem"))
}
