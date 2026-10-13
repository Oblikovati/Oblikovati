// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    EDIT_METHODS, FileQueryHost, TRANSACTION_METHODS, command_owner, decode, encode, host::busy,
    transactions,
};
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_model::{attachment::ExternalAttachmentFiles, identity::GuidSource};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::cell::RefCell;
mod attachments;
mod close;
mod derived_links;
mod selection;
pub use derived_links::DERIVED_COMMAND_METHODS;
mod create;
mod lifecycle;
mod open;
mod sketch_preferences;
pub use open::DocumentObserverFactory;

/// Implemented calls that require the file owner.
pub const FILE_COMMAND_METHODS: [&str; 10] = [
    METHOD_DOCUMENTS_ADD_ATTACHMENT,
    METHOD_DOCUMENTS_ACTIVATE,
    METHOD_DOCUMENTS_OPEN,
    METHOD_DOCUMENTS_CREATE,
    METHOD_DOCUMENTS_BATCH_SAVE,
    METHOD_DOCUMENTS_SAVE,
    METHOD_DOCUMENTS_SAVE_AS,
    METHOD_DOCUMENTS_SAVE_COPY_AS,
    METHOD_DOCUMENTS_CLOSE,
    METHOD_DOCUMENTS_CLOSE_ALL,
];

/// Synchronous API access that retains file, history, and identity ownership.
pub struct FileHost<'a, O: TransactionObserver, W, F> {
    context: RefCell<FileContext<'a, O, W, F>>,
}
struct FileContext<'a, O: TransactionObserver, W, F> {
    files: &'a mut FileWorkspace<O, W, F>,
    source: &'a mut dyn GuidSource,
    version: &'a str,
    external: Option<&'a mut dyn ExternalAttachmentFiles>,
    observers: Option<&'a mut dyn DocumentObserverFactory<Observer = O>>,
    sketch_preferences:
        Option<&'a mut dyn oblikovati_model::application_options::SketchPreferenceAccess>,
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
                observers: None,
                external: None,
                sketch_preferences: None,
            }),
        }
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> Caller
    for FileHost<'_, O, W, F>
{
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut context = self.context.try_borrow_mut().map_err(|_| busy(method))?;
        if crate::CLIENT_COMMAND_METHODS.contains(&method) {
            return crate::clients::apply(context.files.connected_clients_mut(), method, request);
        }
        if method == METHOD_DOCUMENTS_LIST_ATTACHMENTS {
            return context.list_attachments(decode(method, request)?);
        }
        if crate::DATUM_COMMAND_METHODS.contains(&method) {
            return crate::datums::apply(context.files, method, request);
        }
        if crate::SKETCH_COMMAND_METHODS.contains(&method) {
            return context.sketch_command(method, request);
        }
        context.owned_command(method, request)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn owned_command(&mut self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if crate::ASSET_COMMAND_METHODS.contains(&method) {
            return crate::asset_commands::apply(self.files, method, request);
        }
        if crate::SKETCH_OPTION_METHODS.contains(&method) {
            return self.sketch_option_command(method, request);
        }
        if EDIT_METHODS.contains(&method) {
            return command_owner::edit(self.files, method, request);
        }
        if TRANSACTION_METHODS.contains(&method) {
            return transactions::apply(self.files, method, request);
        }
        if DERIVED_COMMAND_METHODS.contains(&method) {
            return self.derived_command(method, request);
        }
        if FILE_COMMAND_METHODS.contains(&method) {
            return self.lifecycle_command(method, request);
        }
        FileQueryHost::new(self.files).call(method, request)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn command(&mut self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
            METHOD_DOCUMENTS_ADD_ATTACHMENT => self.add_attachment(decode(method, request)?),
            METHOD_DOCUMENTS_CREATE => self.create(decode(method, request)?),
            METHOD_DOCUMENTS_OPEN => self.open(decode(method, request)?),
            METHOD_DOCUMENTS_ACTIVATE => self.activate(decode(method, request)?),
            METHOD_DOCUMENTS_BATCH_SAVE => self.batch_save(decode(method, request)?),
            METHOD_DOCUMENTS_SAVE => self.save(decode(method, request)?),
            METHOD_DOCUMENTS_SAVE_AS => self.save_as(decode(method, request)?),
            METHOD_DOCUMENTS_SAVE_COPY_AS => self.save_copy(decode(method, request)?),
            METHOD_DOCUMENTS_CLOSE_ALL => self.close_all(decode(method, request)?),
            METHOD_DOCUMENTS_CLOSE => self.close(decode(method, request)?),
            _ => Err(Error(format!(
                "file command {method:?}: expected one of {FILE_COMMAND_METHODS:?}"
            ))),
        }
    }

    fn batch_save(&mut self, request: BatchSaveArgs) -> Result<Vec<u8>, Error> {
        let response = self
            .files
            .batch_save(request, self.version, self.source)
            .map_err(|error| Error(error.to_string()))?;
        encode(&response)
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

    fn save_as(&mut self, request: SaveDocumentAsArgs) -> Result<Vec<u8>, Error> {
        let response = self
            .files
            .save_as_report(request, self.version, self.source)
            .map_err(|error| Error(error.to_string()))?;
        encode(&response)
    }

    fn save_copy(&mut self, request: SaveCopyAsArgs) -> Result<Vec<u8>, Error> {
        let response = encode(&SaveDocumentResult {
            full_document_name: request.target_file_name.clone(),
        })?;
        self.files
            .save_copy(
                request.document,
                std::path::Path::new(&request.target_file_name),
                request.metadata.as_deref(),
                self.version,
                self.source,
            )
            .map_err(|error| Error(error.to_string()))?;
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
                METHOD_DOCUMENTS_LIST_ATTACHMENTS,
                METHOD_DOCUMENTS_ADD_ATTACHMENT,
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
