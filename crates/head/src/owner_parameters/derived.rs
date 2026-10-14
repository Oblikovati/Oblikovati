// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::GuiParameterLinkEdit;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
pub(in crate::owner) fn require_linked_table_owner<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    table: i64,
) -> Result<(), String> {
    if files.workspace().active_document() != Some(document) {
        return Err(format!(
            "GUI linked-table document {document}: expected current document {:?}",
            files.workspace().active_document()
        ));
    }
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| cause.to_string())?;
    if table <= 0 || session.snapshot().parameters.derived_table(table).is_none() {
        return Err(format!(
            "GUI linked table {table} in document {document}: expected a registered positive table identity"
        ));
    }
    Ok(())
}
pub(in crate::owner) fn linked_source_names<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
    edit: &GuiParameterLinkEdit,
) -> Result<(String, Vec<String>), String> {
    if files.workspace().active_document() != Some(edit.document) || edit.request == 0 {
        return Err(format!(
            "GUI link request {} for document {}: expected the current document and positive request",
            edit.request, edit.document
        ));
    }
    let source = super::owner_link_sources(files, edit.document)?
        .into_iter()
        .find(|source| source.document == edit.source_document)
        .ok_or_else(|| {
            format!(
                "GUI link source {}: expected a current open numeric-user source",
                edit.source_document
            )
        })?;
    let mut names = Vec::with_capacity(edit.linked.len());
    for id in &edit.linked {
        let parameter = source.parameters.iter().find(|parameter| parameter.parameter == *id).ok_or_else(|| format!("GUI link parameter {id} in source {}: expected a current numeric user parameter", edit.source_document))?;
        names.push(parameter.name.clone());
    }
    Ok((source.source_document, names))
}
