// SPDX-License-Identifier: GPL-2.0-only

//! File bindings owned together with their document sessions.
mod assembly_cache_validation;
mod assembly_full_loading;
mod assembly_placement;
mod assembly_sources;
mod asset_catalog;
pub use asset_catalog::WorkspaceAssetCatalogSource;
mod batch;
pub use batch::BatchSaveError;
mod copy;
mod create;
mod derived_edit;
mod discovery;
mod document_references;
mod error;
mod history_restore;
mod name;
pub use name::derive_document_title;
mod open;
mod project;
mod project_assets;
mod projects;
pub use projects::LoadedProject;
mod part_recompute;
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
    assembly_view: Option<(PathBuf, oblikovati_persistence::package::SavedAssemblyView)>,
    express_view: Option<oblikovati_persistence::package::SavedAssemblyView>,
    deferred_views: Option<oblikovati_persistence::package::SavedDrawingViews>,
}
/// A picture for the next save of a file, or none in it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FilePicture {
    Store(oblikovati_persistence::package::SavedPicture),
    Remove,
}
impl RetainedFile {
    fn require_full_content(&self, id: u64) -> Result<(), FileWorkspaceError> {
        if self.express_view.is_some() {
            return Err(FileWorkspaceError::ExpressMode(id, false));
        }
        Ok(())
    }

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
    document_references: BTreeMap<u64, BTreeMap<String, u64>>,
    document_reference_errors: BTreeMap<u64, BTreeMap<String, String>>,
    background_reference_models: std::collections::BTreeSet<u64>,
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
            document_references: BTreeMap::new(),
            document_reference_errors: BTreeMap::new(),
            background_reference_models: std::collections::BTreeSet::new(),
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
        if self
            .retained
            .get(&id)
            .is_some_and(|file| file.express_view.is_some())
        {
            return Err(WorkspaceError::ExpressReadOnly(id));
        }
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

    /// Read the last saved drawing presentation; example: `files.saved_drawing_views(id)?`.
    pub fn saved_drawing_views(
        &self,
        id: u64,
    ) -> Result<Option<oblikovati_persistence::package::SavedDrawingViews>, FileWorkspaceError>
    {
        let file = self.retained_file(id)?;
        file.package
            .restore_saved_drawing_views()
            .map_err(FileWorkspaceError::Package)
    }

    /// Freeze saved drawing geometry for this open session; example: `files.defer_drawing_updates(id)?`.
    pub fn defer_drawing_updates(&mut self, id: u64) -> Result<(), FileWorkspaceError> {
        let views = self.saved_drawing_views(id)?.ok_or_else(|| {
            FileWorkspaceError::DeferredDrawing(id, "expected saved view geometry".into())
        })?;
        let session = self
            .workspace
            .session(id)
            .map_err(FileWorkspaceError::Workspace)?;
        let current = session.snapshot().drawing.as_ref().ok_or_else(|| {
            FileWorkspaceError::DeferredDrawing(id, "expected a drawing document".into())
        })?;
        if current.recipe() != &views.drawing_recipe {
            return Err(FileWorkspaceError::DeferredDrawing(
                id,
                "expected the saved drawing recipe".into(),
            ));
        }
        self.retained
            .get_mut(&id)
            .ok_or(FileWorkspaceError::MissingFile(id))?
            .deferred_views = Some(views);
        Ok(())
    }

    /// Read the frozen presentation, if updates are deferred; example: `files.deferred_drawing_views(id)?`.
    pub fn deferred_drawing_views(
        &self,
        id: u64,
    ) -> Result<Option<&oblikovati_persistence::package::SavedDrawingViews>, FileWorkspaceError>
    {
        Ok(self.retained_file(id)?.deferred_views.as_ref())
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
        let references = self.owned_assembly_references(id);
        let mut target = bound_target(&mut self.retained, &mut self.files, id, references)?;
        self.workspace
            .save(id, version, source, &mut target)
            .map_err(FileWorkspaceError::Save)
    }

    /// Release path and GUID claims only after close policy accepts.
    /// Example: `host.close_discarding_changes(id)?;` after the caller chooses discard.
    pub fn close_discarding_changes(&mut self, id: u64) -> Result<O, FileWorkspaceError> {
        if self.has_reference_dependents(id) {
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
        if self.has_reference_dependents(id) {
            return Err(FileWorkspaceError::ReferencedModel(id));
        }
        if self.retained_file(id)?.express_view.is_some() {
            let session = self
                .workspace
                .session(id)
                .map_err(FileWorkspaceError::Workspace)?;
            if session.in_transaction()
                || self
                    .workspace
                    .needs_save(id)
                    .map_err(FileWorkspaceError::Workspace)?
            {
                return Err(FileWorkspaceError::ExpressMode(id, false));
            }
            return self.close_discarding_changes(id);
        }
        let references = self.owned_assembly_references(id);
        let mut target = bound_target(&mut self.retained, &mut self.files, id, references)?;
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
        self.release_document_references(id);
        self.background_reference_models.remove(&id);
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
    references: std::collections::BTreeSet<String>,
) -> Result<ParameterPackageTarget<'a, F>, FileWorkspaceError> {
    let file = retained
        .get_mut(&id)
        .ok_or(FileWorkspaceError::MissingFile(id))?;
    file.require_full_content(id)?;
    file.apply_picture();
    let path = file
        .path
        .as_deref()
        .ok_or(FileWorkspaceError::UnboundDocument(id))?;
    let name = name::saved_name(file, path);
    let destination = path.to_owned();
    let staged = take_assembly_view(file, &destination)?;
    let path = file.path.as_deref().expect("validated destination");
    Ok(ParameterPackageTarget::new(&mut file.package, path, files)
        .with_display_name(name)
        .with_assembly_references(references)
        .with_drawing_views(file.drawing_views.clone())
        .with_assembly_view(staged))
}

fn take_assembly_view(
    file: &mut RetainedFile,
    destination: &Path,
) -> Result<Option<oblikovati_persistence::package::SavedAssemblyView>, FileWorkspaceError> {
    let Some((expected, view)) = file.assembly_view.take() else {
        return Ok(None);
    };
    if expected != destination {
        return Err(FileWorkspaceError::ReferenceGraph(format!(
            "assembly save destination {destination:?}: expected staged destination {expected:?}"
        )));
    }
    Ok(Some(view))
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Update external connections without exposing mutable document ownership.
    /// Example: `files.connected_clients_mut().register("cam")?;`.
    pub fn connected_clients_mut(&mut self) -> &mut crate::clients::ConnectedClients {
        self.workspace.connected_clients_mut()
    }
}
