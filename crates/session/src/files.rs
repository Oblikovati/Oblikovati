// SPDX-License-Identifier: GPL-2.0-only

//! File bindings owned together with their document sessions.
mod asset_catalog;
pub use asset_catalog::WorkspaceAssetCatalogSource;
mod batch;
pub use batch::BatchSaveError;
mod copy;
mod create;
mod derived_edit;
mod discovery;
mod drawing_references;
mod error;
mod history_restore;
mod name;
pub use name::derive_document_title;
mod open;
mod project;
mod project_assets;
mod projects;
pub use projects::LoadedProject;
mod propagation;
mod refresh;
pub use propagation::{BlockedDerivedDocument, DerivedRefreshFailure, DerivedRefreshReport};
mod save_as;
mod sketch_inference;
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
    /// What the next save stores as the file's picture; nothing asked leaves it as it is.
    picture: Option<FilePicture>,
    drawing_views: Option<oblikovati_persistence::package::SavedDrawingViews>,
}
/// A picture for the next save of a file, or none in it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FilePicture {
    Store(oblikovati_persistence::package::SavedPicture),
    Remove,
}
impl RetainedFile {
    /// A save writes what is latest; keeping it apart stops a redraw encoding a package.
    fn apply_picture(&mut self) {
        match self.picture.take() {
            Some(FilePicture::Store(picture)) => {
                self.package = self.package.with_preview_picture(Some(&picture));
            }
            Some(FilePicture::Remove) => self.package = self.package.with_preview_picture(None),
            None => {}
        }
    }
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
    drawing_references: BTreeMap<u64, (String, u64)>,
    drawing_reference_errors: BTreeMap<u64, (String, String)>,
    drawing_background_models: std::collections::BTreeSet<u64>,
    files: F,
    projects: Vec<LoadedProject>,
    active_project: Option<usize>,
    asset_catalog_source: Box<dyn WorkspaceAssetCatalogSource>,
    asset_catalog:
        std::cell::OnceCell<Result<oblikovati_model::assets::BuiltinAssetCatalog, String>>,
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
        Self::new_with_asset_catalog_source(
            observer,
            files,
            options,
            Box::new(asset_catalog::EmbeddedWorkspaceCatalog),
        )
    }
    /// Inject an immutable catalog source without loading it; example: `FileWorkspace::new_with_asset_catalog_source(observer, files, options, source)`.
    pub fn new_with_asset_catalog_source(
        observer: W,
        files: F,
        options: oblikovati_solve::nonlinear::SolveOptions,
        source: Box<dyn WorkspaceAssetCatalogSource>,
    ) -> Self {
        Self {
            workspace: WorkspaceSession::new_with_sketch_options(observer, options),
            retained: BTreeMap::new(),
            paths: BTreeMap::new(),
            drawing_references: BTreeMap::new(),
            drawing_reference_errors: BTreeMap::new(),
            drawing_background_models: std::collections::BTreeSet::new(),
            files,
            projects: Vec::new(),
            active_project: None,
            asset_catalog_source: source,
            asset_catalog: std::cell::OnceCell::new(),
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

    /// Say what the document's next save stores as its picture; a closed document needs none.
    /// Example: `host.keep_file_picture(id, FilePicture::Remove);`.
    pub fn keep_file_picture(&mut self, id: u64, picture: FilePicture) {
        if let Some(file) = self.retained.get_mut(&id) {
            file.picture = Some(picture);
        }
    }
    /// Supply a drawing presentation for its next save; example: `files.keep_drawing_views(id, views)?`.
    pub fn keep_drawing_views(
        &mut self,
        id: u64,
        views: oblikovati_persistence::package::SavedDrawingViews,
    ) -> Result<(), FileWorkspaceError> {
        let file = self
            .retained
            .get_mut(&id)
            .ok_or(FileWorkspaceError::MissingFile(id))?;
        file.drawing_views = Some(views);
        Ok(())
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
        if self.has_drawing_dependents(id) {
            return Err(FileWorkspaceError::ReferencedModel(id));
        }
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
        if self.has_drawing_dependents(id) {
            return Err(FileWorkspaceError::ReferencedModel(id));
        }
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
        self.release_drawing_reference(id);
        self.drawing_background_models.remove(&id);
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
    file.apply_picture();
    let path = file
        .path
        .as_deref()
        .ok_or(FileWorkspaceError::UnboundDocument(id))?;
    let name = name::saved_name(file, path);
    Ok(ParameterPackageTarget::new(&mut file.package, path, files)
        .with_display_name(name)
        .with_drawing_views(file.drawing_views.clone()))
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Update external connections without exposing mutable document ownership.
    /// Example: `files.connected_clients_mut().register("cam")?;`.
    pub fn connected_clients_mut(&mut self) -> &mut crate::clients::ConnectedClients {
        self.workspace.connected_clients_mut()
    }
}
