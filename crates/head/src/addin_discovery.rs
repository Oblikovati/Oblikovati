// SPDX-License-Identifier: GPL-2.0-only
use std::{
    ffi::{OsStr, OsString},
    io,
    path::{Path, PathBuf},
};
pub(crate) struct AddInDirectoryEntry {
    pub name: OsString,
    pub directory: bool,
}
pub(crate) trait AddInDirectoryReader {
    fn entries(&self, directory: &Path) -> io::Result<Vec<AddInDirectoryEntry>>;
}
pub(crate) struct NativeAddInDirectoryReader;
impl AddInDirectoryReader for NativeAddInDirectoryReader {
    fn entries(&self, directory: &Path) -> io::Result<Vec<AddInDirectoryEntry>> {
        std::fs::read_dir(directory)?
            .map(|entry| {
                let entry = entry?;
                Ok(AddInDirectoryEntry {
                    name: entry.file_name(),
                    directory: entry.file_type()?.is_dir(),
                })
            })
            .collect()
    }
}
pub(crate) fn addin_directory(
    override_path: Option<&OsStr>,
    executable: io::Result<PathBuf>,
) -> PathBuf {
    if let Some(path) = override_path.filter(|path| !path.is_empty()) {
        return path.into();
    }
    executable
        .ok()
        .and_then(|path| path.parent().map(Path::to_path_buf))
        .unwrap_or_default()
        .join("addins")
}
pub(crate) fn discover_addins(
    reader: &impl AddInDirectoryReader,
    directory: &Path,
) -> Result<Vec<PathBuf>, String> {
    let entries = match reader.entries(directory) {
        Ok(entries) => entries,
        Err(cause) if cause.kind() == io::ErrorKind::NotFound => return Ok(Vec::new()),
        Err(cause) => {
            return Err(format!(
                "add-in directory {directory:?}: expected readable entries: {cause}"
            ));
        }
    };
    let mut paths = Vec::new();
    for entry in entries {
        if !entry.directory && shared_library_name(&entry.name) {
            paths.push(directory.join(entry.name));
        }
    }
    paths.sort();
    Ok(paths)
}
fn shared_library_name(name: &OsStr) -> bool {
    Path::new(name)
        .extension()
        .and_then(OsStr::to_str)
        .is_some_and(|extension| {
            ["so", "dll", "dylib"]
                .iter()
                .any(|candidate| extension.eq_ignore_ascii_case(candidate))
        })
}
#[cfg(test)]
#[path = "../tests/addin_discovery.rs"]
mod tests;
