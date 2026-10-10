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
            visible: file.visible,
            active: self.workspace.active_document() == Some(id),
        })
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Change logical visibility without changing focus or the saved recipe.
    /// Example: `host.set_visible(id, false)?` hides a fully loaded document.
    pub fn set_visible(&mut self, id: u64, visible: bool) -> Result<(), FileWorkspaceError> {
        self.retained
            .get_mut(&id)
            .ok_or(FileWorkspaceError::MissingFile(id))?
            .visible = visible;
        Ok(())
    }
}
