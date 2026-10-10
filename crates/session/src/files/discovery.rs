// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::DocumentInfo;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Inspect a file-owned document without activating or loading it again.
    /// Example: `host.document_info(id)?` supplies an API document-list entry.
    pub fn document_info(&self, id: u64) -> Result<DocumentInfo, FileWorkspaceError> {
        let file = self
            .retained
            .get(&id)
            .ok_or(FileWorkspaceError::MissingFile(id))?;
        let manifest = file.package.manifest();
        Ok(DocumentInfo {
            id,
            name: self
                .workspace
                .display_name(id)
                .map_err(FileWorkspaceError::Workspace)?
                .into(),
            type_: manifest.type_name().into(),
            sub_type: manifest.sub_type.clone(),
            dirty: self
                .workspace
                .needs_save(id)
                .map_err(FileWorkspaceError::Workspace)?,
            // FileWorkspace currently owns visible documents only; background open preserves focus.
            visible: true,
            active: self.workspace.active_document() == Some(id),
        })
    }
}
