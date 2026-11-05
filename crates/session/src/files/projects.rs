// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_persistence::{
    project::{ProjectFileLookup, ProjectFileProbe},
    project_file::{BoundProject, ProjectConfiguration, read_project_file},
};

/// Validated project ownership; its canonical path is its session identity.
#[derive(Clone, Debug, PartialEq)]
pub struct LoadedProject {
    pub path: PathBuf,
    pub configuration: ProjectConfiguration,
    pub bound: BoundProject,
    pub assets: oblikovati_model::assets::ProjectAssetLibrary,
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Inspect registered projects in load order; example: `host.projects()` for a selector.
    pub fn projects(&self) -> &[LoadedProject] {
        &self.projects
    }

    /// Inspect the selected configuration; example: `host.active_project().map(|p| &p.bound.lookup)`.
    pub fn active_project(&self) -> Option<&LoadedProject> {
        self.active_project.map(|index| &self.projects[index])
    }

    /// Load once under canonical identity; the first valid project becomes active.
    /// Example: `let canonical = host.load_project(path)?.path.clone();`.
    pub fn load_project(&mut self, path: &Path) -> Result<&LoadedProject, FileWorkspaceError> {
        let canonical = checked_path(path, self.files.canonical_path(path))?;
        if let Some(index) = self
            .projects
            .iter()
            .position(|project| project.path == canonical)
        {
            return Ok(&self.projects[index]);
        }
        let candidate = self.read_project_candidate(canonical)?;
        let index = self.projects.len();
        self.projects.push(candidate);
        self.active_project.get_or_insert(index);
        Ok(&self.projects[index])
    }

    fn read_project_candidate(
        &mut self,
        canonical: PathBuf,
    ) -> Result<LoadedProject, FileWorkspaceError> {
        let configuration = read_project_file(&canonical, &mut self.files)
            .map_err(FileWorkspaceError::ProjectConfiguration)?;
        let bound = configuration
            .bind(&canonical)
            .map_err(FileWorkspaceError::ProjectConfiguration)?;
        let assets = oblikovati_persistence::assets::load_project_asset_library(
            bound.design_data.as_deref(),
            &mut self.files,
        )
        .map_err(FileWorkspaceError::ProjectAssets)?;
        Ok(LoadedProject {
            path: canonical,
            configuration,
            bound,
            assets,
        })
    }

    /// Select an already loaded canonical project without I/O or document mutation.
    /// Example: `host.select_project(&canonical)?;`.
    pub fn select_project(&mut self, canonical: &Path) -> Result<(), FileWorkspaceError> {
        let index = self
            .projects
            .iter()
            .position(|project| project.path == canonical)
            .ok_or_else(|| FileWorkspaceError::UnknownProject(canonical.into()))?;
        self.active_project = Some(index);
        Ok(())
    }

    /// Reload a registered project's libraries without changing selection or documents.
    /// Example: `host.reload_project_assets(&canonical)?;` after an external library edit.
    pub fn reload_project_assets(&mut self, canonical: &Path) -> Result<(), FileWorkspaceError> {
        let index = self
            .projects
            .iter()
            .position(|project| project.path == canonical)
            .ok_or_else(|| FileWorkspaceError::UnknownProject(canonical.into()))?;
        let candidate = oblikovati_persistence::assets::load_project_asset_library(
            self.projects[index].bound.design_data.as_deref(),
            &mut self.files,
        )
        .map_err(FileWorkspaceError::ProjectAssets)?;
        self.projects[index].assets = candidate;
        Ok(())
    }

    pub(super) fn active_lookup(&self) -> Result<&ProjectFileLookup, FileWorkspaceError> {
        self.active_project()
            .map(|project| &project.bound.lookup)
            .ok_or(FileWorkspaceError::MissingProject)
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles + ProjectFileProbe>
    FileWorkspace<O, W, F>
{
    /// Find a referenced file through the selected workspace/library roots.
    /// Example: `let source = host.resolve_project_reference(Path::new("pin.opd"))?;`.
    pub fn resolve_project_reference(
        &mut self,
        reference: &Path,
    ) -> Result<Option<PathBuf>, FileWorkspaceError> {
        let lookup = self.active_lookup()?.clone();
        lookup
            .resolve(reference, &mut self.files)
            .map_err(FileWorkspaceError::Project)
    }

    /// Use the selected project for default templates, preserving explicit metadata precedence.
    /// Example: `host.create_from_active_project_template(kind, &metadata, observer, "alpha", source)?;`.
    pub fn create_from_active_project_template(
        &mut self,
        kind: oblikovati_api::types::DocumentType,
        metadata: &oblikovati_api::types::NewFileMetadata,
        observer: O,
        version: &str,
        source: &mut dyn GuidSource,
    ) -> Result<RegisteredDocument, FileWorkspaceError> {
        let lookup = self.active_lookup()?.clone();
        self.create_from_project_template(&lookup, kind, metadata, observer, version, source)
    }
}
