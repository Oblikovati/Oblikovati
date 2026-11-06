// SPDX-License-Identifier: GPL-2.0-only
//! File-system reads behind the document dialogs, so tests name a fake instead of touching disk.
use std::time::SystemTime;

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct FolderEntry {
    pub name: String,
    pub is_folder: bool,
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct FileFacts {
    pub size: u64,
    pub created: Option<SystemTime>,
    pub modified: Option<SystemTime>,
    pub accessed: Option<SystemTime>,
}
pub(crate) trait GuiFolders {
    /// The folder a new dialog starts in; example: `folders.default_folder()`.
    fn default_folder(&self) -> String;
    /// Folders first, then files, each sorted by name; example: `folders.entries("/tmp")?`.
    fn entries(&self, folder: &str) -> Result<Vec<FolderEntry>, String>;
    /// One file's size and timestamps; example: `folders.file_facts("/tmp/a.opd")?`.
    fn file_facts(&self, path: &str) -> Result<FileFacts, String>;
}
pub(crate) struct NativeFolders;
impl GuiFolders for NativeFolders {
    fn default_folder(&self) -> String {
        std::env::current_dir()
            .map(|path| path.display().to_string())
            .unwrap_or_default()
    }
    fn entries(&self, folder: &str) -> Result<Vec<FolderEntry>, String> {
        let entries = std::fs::read_dir(folder)
            .map_err(|cause| format!("folder {folder:?}: expected a readable folder: {cause}"))?;
        let mut rows: Vec<FolderEntry> = entries.filter_map(Result::ok).map(folder_entry).collect();
        sort_entries(&mut rows);
        Ok(rows)
    }
    fn file_facts(&self, path: &str) -> Result<FileFacts, String> {
        let facts = std::fs::metadata(path)
            .map_err(|cause| format!("file {path:?}: expected a readable file: {cause}"))?;
        Ok(FileFacts {
            size: facts.len(),
            created: facts.created().ok(),
            modified: facts.modified().ok(),
            accessed: facts.accessed().ok(),
        })
    }
}
fn folder_entry(entry: std::fs::DirEntry) -> FolderEntry {
    FolderEntry {
        name: entry.file_name().to_string_lossy().into_owned(),
        is_folder: entry.file_type().is_ok_and(|kind| kind.is_dir()),
    }
}
/// An explicit total order: a folder listing never depends on the platform's read order.
fn sort_entries(rows: &mut [FolderEntry]) {
    rows.sort_by(|left, right| (!left.is_folder, &left.name).cmp(&(!right.is_folder, &right.name)));
}
#[cfg(test)]
#[path = "../../tests/application/file_system.rs"]
mod tests;
