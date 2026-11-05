// SPDX-License-Identifier: GPL-2.0-only
use super::{RecipeIdError, Sketch2D, SketchCopyError, SketchEntityId, SketchError};
use std::collections::BTreeMap;
pub(super) struct CopyIdentities(BTreeMap<SketchEntityId, SketchEntityId>);
impl CopyIdentities {
    pub fn reserve(
        target: &Sketch2D,
        sources: &[SketchEntityId],
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<Self, SketchCopyError> {
        let copies = target.reserve_edit_identities(sources.len(), reserve)?;
        Ok(Self(sources.iter().copied().zip(copies).collect()))
    }
    pub fn get(&self, source: SketchEntityId) -> Result<SketchEntityId, SketchError> {
        self.0.get(&source).copied().ok_or_else(|| SketchError::Definition(format!("copy source {source:?}: expected a reserved identity in the complete copy footprint")))
    }
    pub fn relations(
        &self,
        sources: &[SketchEntityId],
    ) -> Result<Vec<(SketchEntityId, SketchEntityId)>, SketchError> {
        sources
            .iter()
            .map(|source| self.get(*source).map(|copy| (*source, copy)))
            .collect()
    }
    pub fn pair(&self, points: [SketchEntityId; 2]) -> Result<[SketchEntityId; 2], SketchError> {
        Ok([self.get(points[0])?, self.get(points[1])?])
    }
}
