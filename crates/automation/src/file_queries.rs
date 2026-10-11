// SPDX-License-Identifier: GPL-2.0-only

use crate::{SessionQueryHost, encode};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Additional queries that require a file owner, rather than bare recipe sessions.
pub const FILE_QUERY_METHODS: [&str; 9] = [
    METHOD_DOCUMENTS_LIST,
    METHOD_PARAMETERS_DERIVED_TABLES_LIST,
    METHOD_SKETCH_CONSTRAINT_STATUS,
    METHOD_SKETCH_ENTITIES,
    METHOD_SKETCH_REFERENCE_KEY,
    METHOD_SKETCH_RESOLVE_REFERENCE,
    METHOD_WORK_AXES_LIST,
    METHOD_WORK_POINTS_LIST,
    METHOD_WORK_PLANES_LIST,
];
pub(crate) mod derived;

/// Immutable API access to file-owned documents and their session state.
pub struct FileQueryHost<'a, O, W, F> {
    files: &'a FileWorkspace<O, W, F>,
}
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>
    FileQueryHost<'a, O, W, F>
{
    /// Borrow file-owned state; example: `Client::new(FileQueryHost::new(&files))`.
    pub fn new(files: &'a FileWorkspace<O, W, F>) -> Self {
        Self { files }
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> Caller
    for FileQueryHost<'_, O, W, F>
{
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if crate::DATUM_QUERY_METHODS.contains(&method) {
            return crate::datums::query(self.files, method, request);
        }
        if crate::SKETCH_QUERY_METHODS.contains(&method) {
            return crate::sketches::query(self.files, method, request);
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
