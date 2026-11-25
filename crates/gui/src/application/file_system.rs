// SPDX-License-Identifier: GPL-2.0-only
//! File-system reads behind the document dialogs, so tests name a fake instead of touching disk.
use std::time::SystemTime;

#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct FolderEntry {
    pub name: String,
    pub is_folder: bool,
    pub size: u64,
    pub modified: Option<SystemTime>,
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) struct FileFacts {
    pub size: u64,
    pub read_only: bool,
    /// What the file occupies on disk, where the platform reports it.
    pub size_on_disk: Option<u64>,
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
    /// The places a system-location pane offers, as labels and folders.
    fn system_locations(&self) -> Vec<(String, String)>;
}
pub(crate) struct NativeFolders;
impl NativeFolders {
    /// Create one folder under the chosen parent, for example `folders.create_child("/work", "Parts")`.
    pub(crate) fn create_child(&self, parent: &str, name: &str) -> Result<String, String> {
        let parent = std::path::Path::new(parent);
        let name = name.trim();
        if !parent.is_absolute() || !valid_folder_name(name) {
            return Err(format!(
                "folder {name:?} in {parent:?}: expected a single name and an absolute parent"
            ));
        }
        let path = parent.join(name);
        std::fs::create_dir(&path).map_err(|cause| format!("folder {path:?}: {cause}"))?;
        Ok(path.display().to_string())
    }
}

fn valid_folder_name(name: &str) -> bool {
    !name.is_empty()
        && !name.ends_with('.')
        && !name.contains(['/', '\\', ':', '*', '?', '"', '<', '>', '|'])
        && !name.chars().any(char::is_control)
}

impl GuiFolders for NativeFolders {
    fn system_locations(&self) -> Vec<(String, String)> {
        let mut places = Vec::new();
        if let Some(home) = std::env::var_os("HOME") {
            let home = std::path::PathBuf::from(home);
            for (label, folder) in [
                ("Home", home.clone()),
                ("Documents", home.join("Documents")),
            ] {
                if folder.is_dir() {
                    places.push((label.to_string(), folder.display().to_string()));
                }
            }
        }
        places.push(("Computer".to_string(), "/".to_string()));
        places
    }
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
            read_only: facts.permissions().readonly(),
            size_on_disk: disk_size(&facts),
            created: facts.created().ok(),
            modified: facts.modified().ok(),
            accessed: facts.accessed().ok(),
        })
    }
}
/// Allocated blocks are counted in units of 512 bytes, whatever the file system's block size.
#[cfg(unix)]
fn disk_size(facts: &std::fs::Metadata) -> Option<u64> {
    use std::os::unix::fs::MetadataExt;
    Some(facts.blocks() * 512)
}
#[cfg(not(unix))]
fn disk_size(_: &std::fs::Metadata) -> Option<u64> {
    None
}
fn folder_entry(entry: std::fs::DirEntry) -> FolderEntry {
    let facts = entry.metadata().ok();
    FolderEntry {
        name: entry.file_name().to_string_lossy().into_owned(),
        is_folder: entry.file_type().is_ok_and(|kind| kind.is_dir()),
        size: facts.as_ref().map_or(0, std::fs::Metadata::len),
        modified: facts.and_then(|facts| facts.modified().ok()),
    }
}
/// An explicit total order: a folder listing never depends on the platform's read order.
fn sort_entries(rows: &mut [FolderEntry]) {
    rows.sort_by(|left, right| (!left.is_folder, &left.name).cmp(&(!right.is_folder, &right.name)));
}
/// UTC, because no time-zone source is available here; example: `2026-11-06 09:30:00 UTC`.
pub(crate) fn timestamp_text(stamp: SystemTime) -> String {
    let seconds = stamp
        .duration_since(SystemTime::UNIX_EPOCH)
        .map_or(0, |since| since.as_secs());
    let (year, month, day) = civil_date((seconds / 86_400) as i64);
    let (hour, minute, second) = (seconds / 3600 % 24, seconds / 60 % 60, seconds % 60);
    format!("{year:04}-{month:02}-{day:02} {hour:02}:{minute:02}:{second:02} UTC")
}
/// Howard Hinnant's days-to-civil algorithm for the proleptic Gregorian calendar.
fn civil_date(days: i64) -> (i64, u32, u32) {
    let shifted = days + 719_468;
    let era = shifted.div_euclid(146_097);
    let day_of_era = shifted.rem_euclid(146_097);
    let year_of_era =
        (day_of_era - day_of_era / 1460 + day_of_era / 36_524 - day_of_era / 146_096) / 365;
    let day_of_year = day_of_era - (365 * year_of_era + year_of_era / 4 - year_of_era / 100);
    let shifted_month = (5 * day_of_year + 2) / 153;
    let day = (day_of_year - (153 * shifted_month + 2) / 5 + 1) as u32;
    let month = (if shifted_month < 10 {
        shifted_month + 3
    } else {
        shifted_month - 9
    }) as u32;
    let year = year_of_era + era * 400 + i64::from(month <= 2);
    (year, month, day)
}
#[cfg(test)]
#[path = "../../tests/application/file_system.rs"]
mod tests;
