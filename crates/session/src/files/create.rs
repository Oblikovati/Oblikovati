// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{
    DOCUMENT_ASSEMBLY, DOCUMENT_DRAWING, DOCUMENT_PART, DOCUMENT_PRESENTATION, DocumentType,
};
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::FileIdentity,
};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Create an unsaved active document from a validated recipe package, without writing a file.
    /// The package supplies content, not the new document's persisted identity.
    /// Example: `let created = host.create(path, package, observer, "alpha", source)?;`.
    pub fn create(
        &mut self,
        requested: &Path,
        package: DocumentPackage,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, FileWorkspaceError> {
        let destination = self.unowned_destination(requested)?;
        let registration = self.register_package(&package, observer, version, source)?;
        let _outcome = self.install_file(destination, package, registration);
        Ok(registration)
    }

    /// Retain a new document without inventing a save destination.
    /// Example: `host.create_unbound(package, observer, "alpha", source)?`.
    pub fn create_unbound(
        &mut self,
        package: DocumentPackage,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, FileWorkspaceError> {
        let registration = self.register_package(&package, observer, version, source)?;
        self.retained.insert(
            registration.session_id,
            RetainedFile {
                visible: true,
                display_override: Some(package.manifest().display_name.clone()),
                path: None,
                package,
            },
        );
        self.refresh_display_name(registration.session_id);
        Ok(registration)
    }

    fn register_package(
        &mut self,
        package: &DocumentPackage,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, FileWorkspaceError> {
        let history = initial_history(package)?;
        let identity = new_package_identity(package, version, source)?;
        self.workspace
            .register_created(history, identity, observer, source)
            .map_err(FileWorkspaceError::Workspace)
    }
}

fn initial_history(package: &DocumentPackage) -> Result<DocumentEditHistory, FileWorkspaceError> {
    require_document_kind(package.manifest().document_type)?;
    let restored = package
        .restore_document_parameters()
        .map_err(FileWorkspaceError::Package)?;
    let properties = package
        .restore_parameter_properties(&restored.parameters)
        .map_err(FileWorkspaceError::Package)?;
    let snapshot = DocumentSnapshot {
        parameters: restored.parameters,
        properties,
    };
    DocumentEditHistory::new(snapshot, restored.ids).map_err(FileWorkspaceError::History)
}

pub(super) fn require_document_kind(kind: u32) -> Result<(), FileWorkspaceError> {
    if !matches!(
        DocumentType(kind),
        DOCUMENT_PART | DOCUMENT_ASSEMBLY | DOCUMENT_DRAWING | DOCUMENT_PRESENTATION
    ) {
        return Err(FileWorkspaceError::DocumentType(kind));
    }
    Ok(())
}

fn new_package_identity(
    package: &DocumentPackage,
    version: &str,
    source: &mut dyn GuidSource,
) -> Result<FileIdentity, FileWorkspaceError> {
    let record = package
        .identity_record()
        .map_err(FileWorkspaceError::Package)?;
    let fresh = FileIdentity::new(version, source).map_err(FileWorkspaceError::Identity)?;
    if let Some(record) = record.filter(|record| !record.internal_name.is_empty()) {
        let prior = record
            .restore(&fresh)
            .map_err(|error| FileWorkspaceError::TemplateIdentity(error.to_string()))?;
        if prior.internal_name == fresh.internal_name {
            return Err(FileWorkspaceError::TemplateIdentity(
                prior.internal_name.to_string(),
            ));
        }
    }
    Ok(fresh)
}
