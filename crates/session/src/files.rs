// SPDX-License-Identifier: GPL-2.0-only

//! File bindings owned together with their document sessions.
mod batch;
pub use batch::BatchSaveError;
mod copy;
mod create;
mod derived_edit;
mod discovery;
mod error;
mod name;
mod open;
mod project;
mod projects;
pub use projects::LoadedProject;
mod propagation;
mod refresh;
pub use propagation::{BlockedDerivedDocument, DerivedRefreshFailure, DerivedRefreshReport};
mod save_as;
mod source;
mod template;
use crate::{
    TransactionObserver,
    workspace::{
        OpenActivation, RegisteredDocument, SessionCommands, WorkspaceError, WorkspaceObserver,
        WorkspaceSession,
    },
};
pub use error::FileWorkspaceError;
use oblikovati_model::identity::GuidSource;
use oblikovati_persistence::{
    file_access::DocumentFiles,
    package::{DocumentPackage, ParameterPackageTarget},
};
use std::{
    collections::BTreeMap,
    path::{Path, PathBuf},
};

struct RetainedFile {
    visible: bool,
    display_override: Option<String>,
    path: Option<PathBuf>,
    package: DocumentPackage,
}

/// An existing path keeps its live recipe; a new path reports identity registration.
#[must_use]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum FileOpenOutcome {
    Existing(u64),
    Opened(RegisteredDocument),
}
impl FileOpenOutcome {
    /// Read the routing ID for either outcome; example: `let id = outcome.session_id();`.
    pub fn session_id(self) -> u64 {
        match self {
            Self::Existing(id) => id,
            Self::Opened(registration) => registration.session_id,
        }
    }
}

/// Own canonical paths and retained packages; callers cannot substitute a save target.
pub struct FileWorkspace<O, W, F> {
    workspace: WorkspaceSession<O, W>,
    retained: BTreeMap<u64, RetainedFile>,
    paths: BTreeMap<PathBuf, u64>,
    files: F,
    projects: Vec<LoadedProject>,
    active_project: Option<usize>,
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Inject file access and workspace policy; example: `FileWorkspace::new(observer, files)`.
    pub fn new(observer: W, files: F) -> Self {
        Self::new_with_sketch_options(observer, files, Default::default())
    }
    /// Inject bounded sketch policy for open, creation, and subsequent edits.
    /// Example: `FileWorkspace::new_with_sketch_options(observer, files, options)`.
    pub fn new_with_sketch_options(
        observer: W,
        files: F,
        options: oblikovati_solve::nonlinear::SolveOptions,
    ) -> Self {
        Self {
            workspace: WorkspaceSession::new_with_sketch_options(observer, options),
            retained: BTreeMap::new(),
            paths: BTreeMap::new(),
            files,
            projects: Vec::new(),
            active_project: None,
        }
    }

    /// Inspect workspace state without bypassing file ownership; example: `host.workspace().needs_save(id)?`.
    pub fn workspace(&self) -> &WorkspaceSession<O, W> {
        &self.workspace
    }

    /// Inspect file access observations; example: `host.files().read_count()`.
    pub fn files(&self) -> &F {
        &self.files
    }

    /// Inspect a document's bound destination; example: `host.path(id)?`.
    pub fn path(&self, id: u64) -> Result<&Path, FileWorkspaceError> {
        self.retained_file(id)?
            .path
            .as_deref()
            .ok_or(FileWorkspaceError::UnboundDocument(id))
    }

    /// Route edits without allowing replacement of the document owner; example: `host.commands(id)?.undo()?`.
    pub fn commands(&mut self, id: u64) -> Result<SessionCommands<'_, O>, WorkspaceError> {
        self.workspace.commands(id)
    }

    /// Change focus without changing command ownership; example: `host.activate(id)?;`.
    pub fn activate(&mut self, id: u64) -> Result<(), WorkspaceError> {
        self.workspace.activate(id)
    }

    /// Save to this document's owned path and package; example: `host.save(id, "alpha", source)?;`.
    pub fn save(
        &mut self,
        id: u64,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<(), FileWorkspaceError> {
        let mut target = bound_target(&mut self.retained, &mut self.files, id)?;
        self.workspace
            .save(id, version, source, &mut target)
            .map_err(FileWorkspaceError::Save)
    }

    /// Release path and GUID claims only after close policy accepts.
    /// Example: `host.close_discarding_changes(id)?;` after the caller chooses discard.
    pub fn close_discarding_changes(&mut self, id: u64) -> Result<O, FileWorkspaceError> {
        let observer = self
            .workspace
            .close_discarding_changes(id)
            .map_err(FileWorkspaceError::Workspace)?;
        self.release_file(id);
        Ok(observer)
    }

    /// Save dirty committed state, then close; example: `host.close_saving_changes(id, "alpha", source)?;`.
    pub fn close_saving_changes(
        &mut self,
        id: u64,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<O, FileWorkspaceError> {
        let mut target = bound_target(&mut self.retained, &mut self.files, id)?;
        let observer = self
            .workspace
            .close_saving_changes(id, version, source, &mut target)
            .map_err(FileWorkspaceError::Save)?;
        self.release_file(id);
        Ok(observer)
    }

    fn retained_file(&self, id: u64) -> Result<&RetainedFile, FileWorkspaceError> {
        self.retained
            .get(&id)
            .ok_or(FileWorkspaceError::MissingFile(id))
    }

    fn release_file(&mut self, id: u64) {
        let file = self.retained.remove(&id).expect("registered file binding");
        if let Some(path) = file.path {
            self.paths.remove(&path);
        }
    }

    fn unowned_destination(&mut self, requested: &Path) -> Result<PathBuf, FileWorkspaceError> {
        let destination = checked_path(requested, self.files.save_path(requested))?;
        if let Some(&owner) = self.paths.get(&destination) {
            return Err(FileWorkspaceError::DestinationInUse {
                path: destination,
                owner,
            });
        }
        Ok(destination)
    }
}

fn checked_path(
    requested: &Path,
    result: std::io::Result<PathBuf>,
) -> Result<PathBuf, FileWorkspaceError> {
    let canonical = result.map_err(|cause| FileWorkspaceError::Path {
        requested: requested.into(),
        cause,
    })?;
    if !canonical.is_absolute() {
        return Err(FileWorkspaceError::RelativeCanonical(canonical));
    }
    Ok(canonical)
}

fn bound_target<'a, F: DocumentFiles>(
    retained: &'a mut BTreeMap<u64, RetainedFile>,
    files: &'a mut F,
    id: u64,
) -> Result<ParameterPackageTarget<'a, F>, FileWorkspaceError> {
    let file = retained
        .get_mut(&id)
        .ok_or(FileWorkspaceError::MissingFile(id))?;
    Ok(ParameterPackageTarget::new(
        &mut file.package,
        file.path
            .as_deref()
            .ok_or(FileWorkspaceError::UnboundDocument(id))?,
        files,
    ))
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Update external connections without exposing mutable document ownership.
    /// Example: `files.connected_clients_mut().register("cam")?;`.
    pub fn connected_clients_mut(&mut self) -> &mut crate::clients::ConnectedClients {
        self.workspace.connected_clients_mut()
    }
}
