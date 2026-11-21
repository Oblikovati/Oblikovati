// SPDX-License-Identifier: GPL-2.0-only
//! Files the application opened or saved, newest first, kept with the application options.

/// One file on the list; its other facts are read from the file when shown.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct RecentDocument {
    pub path: String,
    /// Seconds since the Unix epoch.
    pub last_opened: u64,
    #[serde(default)]
    pub pinned: bool,
}
/// How many files the list keeps; older ones drop off the end.
pub const RECENT_DOCUMENT_LIMIT: usize = 50;

/// Put a file at the head of the list, once; example: after an open or a save.
pub fn note_recent_document(list: &mut Vec<RecentDocument>, path: &str, now: u64) {
    let pinned = list.iter().any(|known| known.path == path && known.pinned);
    list.retain(|known| known.path != path);
    list.insert(
        0,
        RecentDocument {
            path: path.to_owned(),
            last_opened: now,
            pinned,
        },
    );
    list.truncate(RECENT_DOCUMENT_LIMIT);
}
/// Inject the one list owner across the session; failure must keep the list in effect.
pub trait RecentDocumentAccess {
    /// Read the stored list; example: `store.recent_documents()`.
    fn recent_documents(&self) -> Vec<RecentDocument>;
    /// Persist the whole list; example: `store.set_recent_documents(list)?`.
    fn set_recent_documents(&mut self, list: Vec<RecentDocument>) -> Result<(), String>;
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn a_noted_file_moves_to_the_head_once_and_the_list_stays_bounded() {
        let mut list = Vec::new();
        note_recent_document(&mut list, "a", 1);
        note_recent_document(&mut list, "b", 2);
        note_recent_document(&mut list, "a", 3);
        assert_eq!(
            list[0],
            RecentDocument {
                path: "a".into(),
                last_opened: 3,
                pinned: false
            }
        );
        assert_eq!(list.len(), 2);
        for index in 0..RECENT_DOCUMENT_LIMIT + 5 {
            note_recent_document(&mut list, &index.to_string(), 4);
        }
        assert_eq!(list.len(), RECENT_DOCUMENT_LIMIT);
    }
}
