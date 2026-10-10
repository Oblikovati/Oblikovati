// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_model::history::{DocumentEditHistory, DocumentSnapshot};
use oblikovati_persistence::load::{PreparedParameterPackage, read_parameter_package};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Reuse an open canonical path or validate and register a new file.
    /// Example: `host.open(path, observer, "alpha", source, OpenActivation::Activate)?;`.
    pub fn open(
        &mut self,
        path: &Path,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        self.open_with_factory(path, || Ok(observer), version, source, activation)
    }

    /// Create an observer only for a new canonical document owner.
    /// Example: `host.open_with_factory(path, || factory.create(), "alpha", source, activation)?`.
    pub fn open_with_factory(
        &mut self,
        path: &Path,
        factory: impl FnOnce() -> Result<O, String>,
        version: &str,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        if let Some(&id) = self.paths.get(path) {
            return self.reuse_open(id, activation);
        }
        let canonical = checked_path(path, self.files.canonical_path(path))?;
        if let Some(&id) = self.paths.get(&canonical) {
            return self.reuse_open(id, activation);
        }
        let observer = factory().map_err(FileWorkspaceError::ObserverFactory)?;
        self.open_new(canonical, observer, version, source, activation)
    }

    fn reuse_open(
        &mut self,
        id: u64,
        activation: OpenActivation,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        if activation == OpenActivation::Activate {
            self.workspace
                .activate(id)
                .map_err(FileWorkspaceError::Workspace)?;
        }
        Ok(FileOpenOutcome::Existing(id))
    }

    fn open_new(
        &mut self,
        path: PathBuf,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
        activation: OpenActivation,
    ) -> Result<FileOpenOutcome, FileWorkspaceError> {
        let prepared = self.prepare_open(&path, version, source)?;
        let snapshot = DocumentSnapshot {
            parameters: prepared.restored.parameters,
            properties: prepared.properties,
        };
        let history = DocumentEditHistory::new(snapshot, prepared.restored.ids)
            .map_err(FileWorkspaceError::History)?;
        let registration = self
            .workspace
            .register(history, prepared.identity, observer, source, activation)
            .map_err(FileWorkspaceError::Workspace)?;
        Ok(self.install_file(path, prepared.package, registration))
    }

    fn prepare_open(
        &mut self,
        path: &Path,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<PreparedParameterPackage, FileWorkspaceError> {
        self.workspace
            .approve_open(path)
            .map_err(|reason| FileWorkspaceError::OpenVeto {
                path: path.into(),
                reason,
            })?;
        read_parameter_package(path, version, source, &mut self.files)
            .map_err(FileWorkspaceError::Load)
    }

    pub(super) fn install_file(
        &mut self,
        path: PathBuf,
        package: DocumentPackage,
        registration: RegisteredDocument,
    ) -> FileOpenOutcome {
        let display_override = name::display_override(&path, &package.manifest().display_name);
        let package = if display_override.is_none() {
            package.with_display_name("")
        } else {
            package
        };
        self.paths.insert(path.clone(), registration.session_id);
        self.retained.insert(
            registration.session_id,
            RetainedFile {
                visible: true,
                path: Some(path),
                package,
                display_override,
            },
        );
        self.refresh_display_name(registration.session_id);
        FileOpenOutcome::Opened(registration)
    }
}
