// SPDX-License-Identifier: GPL-2.0-only

use super::*;

pub(super) fn display_override(path: &Path, persisted: &str) -> Option<String> {
    // Go storage can contain a derived title. Such a title must follow later Save As paths.
    (!persisted.is_empty() && persisted != derive_document_title(path)).then(|| persisted.into())
}

/// Derive the filename title using Go naming rules; example: `derive_document_title(Path::new("Bracket.opd"))` returns `"Bracket"`.
pub fn derive_document_title(path: &Path) -> String {
    let base = path.file_name().unwrap_or_default().to_string_lossy();
    base.rsplit_once('.')
        .map_or(base.as_ref(), |(stem, _)| stem)
        .into()
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    pub(super) fn refresh_display_name(&mut self, id: u64) {
        let file = &self.retained[&id];
        let name = saved_name(file, file.path.as_deref().unwrap_or(Path::new("")));
        self.workspace.set_file_display_name(id, name);
    }
}

pub(super) fn saved_name(file: &super::RetainedFile, destination: &Path) -> String {
    file.display_override
        .clone()
        .unwrap_or_else(|| derive_document_title(destination))
}
