// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    EDIT_METHODS, FileQueryHost, TRANSACTION_METHODS, command_owner, decode, encode, host::busy,
    transactions,
};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::cell::RefCell;

/// Implemented lifecycle calls that require the file owner.
pub const FILE_COMMAND_METHODS: [&str; 3] = [
    METHOD_DOCUMENTS_ACTIVATE,
    METHOD_DOCUMENTS_SAVE,
    METHOD_DOCUMENTS_CLOSE,
];

/// Synchronous API access that retains file, history, and identity ownership.
pub struct FileHost<'a, O, W, F> {
    context: RefCell<FileContext<'a, O, W, F>>,
}
struct FileContext<'a, O, W, F> {
    files: &'a mut FileWorkspace<O, W, F>,
    source: &'a mut dyn GuidSource,
    version: &'a str,
}
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Borrow owned files and identity generation; example: `FileHost::new(&mut files, &mut guids, "alpha")`.
    pub fn new(
        files: &'a mut FileWorkspace<O, W, F>,
        source: &'a mut dyn GuidSource,
        version: &'a str,
    ) -> Self {
        Self {
            context: RefCell::new(FileContext {
                files,
                source,
                version,
            }),
        }
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> Caller
    for FileHost<'_, O, W, F>
{
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut context = self.context.try_borrow_mut().map_err(|_| busy(method))?;
        if EDIT_METHODS.contains(&method) {
            return command_owner::edit(context.files, method, request);
        }
        if TRANSACTION_METHODS.contains(&method) {
            return transactions::apply(context.files, method, request);
        }
        if FILE_COMMAND_METHODS.contains(&method) {
            return context.command(method, request);
        }
        FileQueryHost::new(context.files).call(method, request)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn command(&mut self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
            METHOD_DOCUMENTS_ACTIVATE => self.activate(decode(method, request)?),
            METHOD_DOCUMENTS_SAVE => self.save(decode(method, request)?),
            METHOD_DOCUMENTS_CLOSE => self.close(decode(method, request)?),
            _ => Err(Error(format!(
                "file command {method:?}: expected one of {FILE_COMMAND_METHODS:?}"
            ))),
        }
    }

    fn activate(&mut self, request: ActivateDocumentArgs) -> Result<Vec<u8>, Error> {
        let response = encode(&OKResult { ok: true })?;
        self.files
            .activate(request.id)
            .map_err(|error| Error(error.to_string()))?;
        Ok(response)
    }

    fn save(&mut self, request: SaveDocumentArgs) -> Result<Vec<u8>, Error> {
        let path = self
            .files
            .path(request.document)
            .map_err(|error| Error(error.to_string()))?;
        let name = path.to_str().ok_or_else(|| {
            Error(format!(
                "save path {path:?}: expected a UTF-8 path for the canonical API response"
            ))
        })?;
        let response = encode(&SaveDocumentResult {
            full_document_name: name.into(),
        })?;
        self.files
            .save(request.document, self.version, self.source)
            .map_err(|error| Error(error.to_string()))?;
        Ok(response)
    }

    fn close(&mut self, request: CloseDocumentArgs) -> Result<Vec<u8>, Error> {
        let response = encode(&CloseDocumentsResult { closed: 1 })?;
        if request.force {
            self.files
                .close_discarding_changes(request.id)
                .map_err(|error| Error(error.to_string()))?;
        } else {
            self.files
                .close_saving_changes(request.id, self.version, self.source)
                .map_err(|error| Error(error.to_string()))?;
        }
        Ok(response)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::host::tests::RejectUnexpectedCalls;
    use oblikovati_model::identity::{DocumentGuid, IdentityTransitionError};
    use oblikovati_persistence::file_access::NativeDocumentFiles;

    struct RejectGuidRequests;
    impl GuidSource for RejectGuidRequests {
        fn generate(&mut self) -> Result<DocumentGuid, IdentityTransitionError> {
            panic!("unexpected GUID request")
        }
    }

    #[test]
    fn file_host_refuses_reentry_and_recovers_for_queries() {
        let mut files = FileWorkspace::<RejectUnexpectedCalls, _, _>::new(
            RejectUnexpectedCalls,
            NativeDocumentFiles,
        );
        let mut source = RejectGuidRequests;
        let host = FileHost::new(&mut files, &mut source, "test");
        {
            let _active = host.context.borrow_mut();
            for method in [
                METHOD_DOCUMENTS_LIST,
                METHOD_DOCUMENTS_SAVE,
                METHOD_PARAMETERS_SET,
                METHOD_TRANSACTION_BEGIN,
            ] {
                assert!(
                    host.call(method, b"{}")
                        .unwrap_err()
                        .to_string()
                        .contains("reentry is refused")
                );
            }
        }
        let reply: ListDocumentsResult =
            serde_json::from_slice(&host.call(METHOD_DOCUMENTS_LIST, b"").unwrap()).unwrap();
        assert_eq!(reply.documents, Some(vec![]));
    }
}
