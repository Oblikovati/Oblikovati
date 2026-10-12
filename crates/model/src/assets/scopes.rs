// SPDX-License-Identifier: GPL-2.0-only
use std::collections::BTreeMap;
/// Stable API identity shared by physical and visual assets; example: `asset.asset_id()`.
pub trait AssetIdentity {
    /// Borrow the stable identity; example: `let id = asset.asset_id();`.
    fn asset_id(&self) -> &str;
}
/// One explicit scope, owning snapshots rather than aliases to another library.
#[derive(Clone, Debug)]
pub struct AssetCollection<T> {
    entries: BTreeMap<String, T>,
}
impl<T> Default for AssetCollection<T> {
    fn default() -> Self {
        Self {
            entries: BTreeMap::new(),
        }
    }
}
impl<T: AssetIdentity> AssetCollection<T> {
    /// Insert an owned asset, returning the old snapshot; example: `scope.insert(asset)`.
    pub fn insert(&mut self, asset: T) -> Option<T> {
        self.entries.insert(asset.asset_id().into(), asset)
    }
    /// Borrow one asset by exact identity; example: `scope.get("steel")`.
    pub fn get(&self, id: &str) -> Option<&T> {
        self.entries.get(id)
    }
    /// Read snapshots in stable ID order; example: `scope.ordered().collect::<Vec<_>>()`.
    pub fn ordered(&self) -> impl Iterator<Item = &T> {
        self.entries.values()
    }
}
/// Document snapshots precede project definitions, which precede shipped definitions.
pub struct ScopedAssets<'a, T> {
    pub document: &'a AssetCollection<T>,
    pub project: &'a AssetCollection<T>,
    pub builtin: &'a AssetCollection<T>,
}
impl<T: AssetIdentity> ScopedAssets<'_, T> {
    /// Resolve the exact ID without rebinding missing assets; example: `scopes.resolve("steel")?`.
    pub fn resolve(&self, id: &str) -> Result<&T, String> {
        [self.document, self.project, self.builtin]
            .into_iter()
            .find_map(|scope| scope.get(id))
            .ok_or_else(|| {
                format!("asset {id:?}: expected an identity in document, project, or builtin scope")
            })
    }
}
