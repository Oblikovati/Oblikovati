// SPDX-License-Identifier: GPL-2.0-only

//! File bindings owned together with their document sessions.
mod batch;
pub use batch::BatchSaveError;
mod copy;
mod create;
mod discovery;
mod error;
mod name;
mod open;
mod project;
mod save_as;
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
    path: PathBuf,
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
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Inject file access and workspace policy; example: `FileWorkspace::new(observer, files)`.
    pub fn new(observer: W, files: F) -> Self {
        Self {
            workspace: WorkspaceSession::new(observer),
            retained: BTreeMap::new(),
            paths: BTreeMap::new(),
            files,
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
        self.retained
            .get(&id)
            .map(|file| file.path.as_path())
            .ok_or(FileWorkspaceError::MissingFile(id))
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

    fn release_file(&mut self, id: u64) {
        let file = self.retained.remove(&id).expect("registered file binding");
        self.paths.remove(&file.path);
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
        &file.path,
        files,
    ))
}
