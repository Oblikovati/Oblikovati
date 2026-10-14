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
mod sketch_construction;
mod sketch_preferences;
pub use open::DocumentObserverFactory;
pub(crate) use sketch_construction::SketchConstructionAccess;

/// Implemented calls that require the file owner.
pub const FILE_COMMAND_METHODS: [&str; 12] = [
    METHOD_DOCUMENT_SET_END_OF_PART,
    METHOD_DOCUMENT_SET_SKETCH_SETTINGS,
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
/// Application preferences that require the injected persistent preference owner.
pub const APPLICATION_PREFERENCE_METHODS: [&str; 2] = [
    METHOD_APPLICATION_GET_HUD_OPTIONS,
    METHOD_APPLICATION_SET_HUD_OPTIONS,
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
    sketch_construction: Option<SketchConstructionAccess<'a>>,
    part_topology: Option<&'a mut dyn crate::PartTopologyAccess<O, W, F>>,
    sketch_preferences:
        Option<&'a mut dyn oblikovati_model::application_options::SketchPreferenceAccess>,
}
impl<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileHost<'a, O, W, F> {
    /// Share the native exact-reference owner; example: `host.with_part_topology(&mut scenes)`.
    pub fn with_part_topology(
        mut self,
        references: &'a mut dyn crate::PartTopologyAccess<O, W, F>,
    ) -> Self {
        self.context.get_mut().part_topology = Some(references);
        self
    }
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
                part_topology: None,
                sketch_construction: None,
            }),
        }
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> Caller
    for FileHost<'_, O, W, F>
{
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut context = self.context.try_borrow_mut().map_err(|_| busy(method))?;
        if APPLICATION_PREFERENCE_METHODS.contains(&method) {
            return context.hud_options(method, request);
        }
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
        if method == METHOD_DOCUMENT_SET_SKETCH_SETTINGS {
            return crate::document_sketch_settings::apply(context.files, decode(method, request)?);
        }
        if method == METHOD_DOCUMENT_SET_END_OF_PART {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::end_of_part::apply(files, part_topology, request);
        }
        if method == METHOD_FEATURES_GET {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::feature_lifecycle::query(files, part_topology, request);
        }
        if method == METHOD_FEATURES_DELETE {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::feature_lifecycle::delete(files, part_topology, request);
        }
        if method == METHOD_MODEL_TREE {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::model_tree::query(files, part_topology, request);
        }
        if crate::FEATURE_LIFECYCLE_METHODS.contains(&method) {
            let FileContext {
                files,
                part_topology,
                ..
            } = &mut *context;
            return crate::feature_lifecycle::apply(files, part_topology, method, request);
        }
        context.owned_command(method, request)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    fn owned_command(&mut self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        if crate::BODY_COMMAND_METHODS.contains(&method) {
            return crate::body_commands::apply(
                self.files,
                &mut self.part_topology,
                method,
                request,
            );
        }
        if crate::BODY_QUERY_METHODS.contains(&method) {
            return crate::part_topology::query(
                self.files,
                &mut self.part_topology,
                method,
                request,
            );
        }
        if crate::ASSET_ASSIGNMENT_METHODS.contains(&method) {
            return crate::asset_assignments::apply(
                self.files,
                &mut self.part_topology,
                method,
                request,
            );
        }
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
