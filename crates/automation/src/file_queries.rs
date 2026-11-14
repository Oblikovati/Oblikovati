// SPDX-License-Identifier: GPL-2.0-only

use crate::{SessionQueryHost, encode};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Additional queries that require a file owner, rather than bare recipe sessions.
pub const FILE_QUERY_METHODS: [&str; 14] = [
    METHOD_ASSEMBLY_OCCURRENCES,
    METHOD_SKETCH_LIST,
    METHOD_SKETCH_GET,
    METHOD_DOCUMENTS_LIST,
    METHOD_PARAMETERS_DERIVED_TABLES_LIST,
    METHOD_SKETCH_CONSTRAINT_STATUS,
    METHOD_SKETCH_CONSTRAINTS,
    METHOD_SKETCH_DIMENSIONS,
    METHOD_SKETCH_ENTITIES,
    METHOD_SKETCH_REFERENCE_KEY,
    METHOD_SKETCH_RESOLVE_REFERENCE,
    METHOD_WORK_AXES_LIST,
    METHOD_WORK_POINTS_LIST,
    METHOD_WORK_PLANES_LIST,
];
pub(crate) mod assembly;
pub(crate) mod derived;

/// Immutable API access to file-owned documents and their session state.
pub struct FileQueryHost<'a, O, W, F> {
    files: &'a FileWorkspace<O, W, F>,
    sketch_edit: Option<crate::SketchInspectionEdit>,
}
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>
    FileQueryHost<'a, O, W, F>
{
    /// Borrow file-owned state; example: `Client::new(FileQueryHost::new(&files))`.
    pub fn new(files: &'a FileWorkspace<O, W, F>) -> Self {
        Self {
            files,
            sketch_edit: None,
        }
    }
    /// Supply the live desktop edit guards; example: `host.with_sketch_edit(Some(edit.clone()))`.
    pub fn with_sketch_edit(mut self, edit: Option<crate::SketchInspectionEdit>) -> Self {
        self.sketch_edit = edit;
        self
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> Caller
    for FileQueryHost<'_, O, W, F>
{
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if method == METHOD_ASSEMBLY_OCCURRENCES {
            let _: oblikovati_api::json::EmptyObject = crate::decode(method, request)?;
            return encode(&assembly::occurrences(self.files)?);
        }
        if crate::ASSET_QUERY_METHODS.contains(&method) {
            return crate::assets::query(self.files, method, request);
        }
        if crate::DATUM_QUERY_METHODS.contains(&method) {
            return crate::datums::query(self.files, method, request);
        }
        if crate::SKETCH_QUERY_METHODS.contains(&method) {
            let editing = self
                .sketch_edit
                .as_ref()
                .and_then(|edit| edit.current(self.files.workspace()));
            let focused = self
                .sketch_edit
                .as_ref()
                .and_then(|edit| edit.focused_identity(self.files.workspace()));
            let editing = pending_sketch_edit(self.files.workspace(), editing, focused);
            return crate::sketches::query(self.files, method, request, editing);
        }
        if method == METHOD_PARAMETERS_DERIVED_TABLES_LIST {
            return encode(&derived::list(self.files)?);
        }
        if method != METHOD_DOCUMENTS_LIST {
            return SessionQueryHost::new(self.files.workspace()).call(method, request);
        }
        encode(&ListDocumentsResult {
            documents: Some(self.documents()?),
        })
    }
}
fn pending_sketch_edit<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &oblikovati_session::workspace::WorkspaceSession<O, W>,
    editing: Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)>,
    focused: Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)>,
) -> Option<(u64, oblikovati_model::document_sketch::DocumentSketchId)> {
    let document = workspace.active_document()?;
    let Some(navigation) = workspace
        .session(document)
        .ok()
        .and_then(|session| session.pending_sketch_navigation())
    else {
        return editing;
    };
    let current = editing
        .filter(|(owner, _)| *owner == document)
        .map(|(_, sketch)| sketch);
    let focused = focused
        .filter(|(owner, _)| *owner == document)
        .map(|(_, sketch)| sketch);
    navigation
        .target(current, focused)
        .map_or(editing, |target| target.map(|sketch| (document, sketch)))
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileQueryHost<'_, O, W, F> {
    fn documents(&self) -> Result<Vec<DocumentInfo>, Error> {
        self.files
            .workspace()
            .documents()
            .iter()
            .map(|id| {
                self.files
                    .document_info(*id)
                    .map_err(|error| Error(error.to_string()))
            })
            .collect()
    }
}
