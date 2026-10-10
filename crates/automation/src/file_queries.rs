// SPDX-License-Identifier: GPL-2.0-only

use crate::{SessionQueryHost, encode};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Additional queries that require a file owner, rather than bare recipe sessions.
pub const FILE_QUERY_METHODS: [&str; 2] =
    [METHOD_DOCUMENTS_LIST, METHOD_PARAMETERS_DERIVED_TABLES_LIST];
mod derived;

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
        if method == METHOD_PARAMETERS_DERIVED_TABLES_LIST {
            return encode(&derived::list(self.files)?);
        }
        if method != METHOD_DOCUMENTS_LIST {
            return SessionQueryHost::new(self.files.workspace()).call(method, request);
        }
        let documents = self
            .files
            .workspace()
            .documents()
            .iter()
            .map(|id| {
                self.files
                    .document_info(*id)
                    .map_err(|error| Error(error.to_string()))
            })
            .collect::<Result<Vec<_>, _>>()?;
        encode(&ListDocumentsResult {
            documents: Some(documents),
        })
    }
}
