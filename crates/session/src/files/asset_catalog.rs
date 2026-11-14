// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::assets::BuiltinAssetCatalog;
/// One injected source supplies the session's immutable shipped asset snapshot.
pub trait WorkspaceAssetCatalogSource: Send + Sync {
    /// Load or name a failed catalog; example: `source.load_catalog()?`.
    fn load_catalog(&self) -> Result<BuiltinAssetCatalog, String>;
}
pub(super) struct EmbeddedWorkspaceCatalog;
impl WorkspaceAssetCatalogSource for EmbeddedWorkspaceCatalog {
    fn load_catalog(&self) -> Result<BuiltinAssetCatalog, String> {
        oblikovati_persistence::assets::load_builtin_catalog()
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Borrow one lazily admitted catalog, retaining a failure too; example: `files.builtin_catalog()?.materials()`.
    pub fn builtin_catalog(&self) -> Result<&BuiltinAssetCatalog, String> {
        self.asset_catalog
            .get_or_init(|| self.asset_catalog_source.load_catalog())
            .as_ref()
            .map_err(Clone::clone)
    }
}
