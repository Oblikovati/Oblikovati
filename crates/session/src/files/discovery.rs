// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::wire::DocumentInfo;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Read the retained package's API kind without assuming recipe contents; example: `host.document_type(id)?`.
    pub fn document_type(
        &self,
        id: u64,
    ) -> Result<oblikovati_api::types::DocumentType, FileWorkspaceError> {
        let file = self
            .retained
            .get(&id)
            .ok_or(FileWorkspaceError::MissingFile(id))?;
        Ok(oblikovati_api::types::DocumentType(
            file.package.manifest().document_type,
        ))
    }
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

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Read current body names and retained legacy styles without changing file state; example: `files.body_display_metadata(document, key)?`.
    pub fn body_display_metadata(
        &self,
        document: u64,
        key: &str,
    ) -> Result<(Option<&str>, Option<&str>), FileWorkspaceError> {
        let name = self
            .workspace
            .session(document)
            .map_err(FileWorkspaceError::Workspace)?
            .snapshot()
            .body_names
            .get(key);
        let (_, style) = self
            .retained
            .get(&document)
            .ok_or(FileWorkspaceError::MissingFile(document))?
            .package
            .body_display_metadata(key)
            .map_err(FileWorkspaceError::Package)?;
        Ok((name.map(String::as_str), style))
    }
}
