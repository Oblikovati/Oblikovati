// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assets::{AppearanceAsset, MaterialAsset, ProjectAssetLibrary};
use oblikovati_persistence::assets::{ProjectAssetFamily, persist_project_asset_family};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Persist a physical project snapshot before adoption; example: `host.store_project_material(material)?`.
    pub fn store_project_material(
        &mut self,
        asset: MaterialAsset,
    ) -> Result<(), FileWorkspaceError> {
        let candidate = self
            .active_project()
            .ok_or(FileWorkspaceError::MissingProject)?
            .assets
            .with_material(asset)
            .map_err(FileWorkspaceError::ProjectAssets)?;
        self.persist_active_project_assets(ProjectAssetFamily::Material, candidate)
    }

    /// Persist a visual project snapshot before adoption; example: `host.store_project_appearance(appearance)?`.
    pub fn store_project_appearance(
        &mut self,
        asset: AppearanceAsset,
    ) -> Result<(), FileWorkspaceError> {
        let candidate = self
            .active_project()
            .ok_or(FileWorkspaceError::MissingProject)?
            .assets
            .with_appearance(asset)
            .map_err(FileWorkspaceError::ProjectAssets)?;
        self.persist_active_project_assets(ProjectAssetFamily::Appearance, candidate)
    }

    fn persist_active_project_assets(
        &mut self,
        family: ProjectAssetFamily,
        candidate: ProjectAssetLibrary,
    ) -> Result<(), FileWorkspaceError> {
        let index = self
            .active_project
            .ok_or(FileWorkspaceError::MissingProject)?;
        let project = &self.projects[index];
        let directory = project.bound.design_data.as_deref().ok_or_else(|| {
            FileWorkspaceError::ProjectAssets(format!(
                "project {:?}: expected an explicit design-data directory for persistence",
                project.path
            ))
        })?;
        persist_project_asset_family(
            directory,
            family,
            &project.assets,
            &candidate,
            &mut self.files,
        )
        .map_err(FileWorkspaceError::ProjectAssets)?;
        self.projects[index].assets = candidate;
        Ok(())
    }
}
