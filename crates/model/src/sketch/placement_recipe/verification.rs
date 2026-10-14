// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{
    CircumcircleOptions, Sketch2D, SketchCircleConstruction, SketchEntityId, SketchReadIndex,
    path_member::indexed_entity,
};
/// Original construction inputs retained until preview/commit verification; no permanent IDs.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct SketchCircularPlacementCheck {
    pub member: usize,
    pub points: [[f64; 2]; 3],
    pub options: CircumcircleOptions,
}
impl SketchPlacementRecipe {
    /// Verify the actual solved members with the injected service; example: `recipe.verify_circular_seeds(&geometry,&members,Some(backend))?`.
    pub fn verify_circular_seeds(
        &self,
        geometry: &Sketch2D,
        members: &[SketchEntityId],
        construction: Option<&dyn SketchCircleConstruction>,
    ) -> Result<(), SketchError> {
        if self.circular_checks.is_empty() {
            return Ok(());
        }
        let backend = construction.ok_or_else(|| {
            SketchError::Definition(format!(
                "placement circular checks {}: expected an injected construction service",
                self.circular_checks.len()
            ))
        })?;
        let index = geometry.read_index();
        for check in &self.circular_checks {
            check.verify(&index, members, backend)?;
        }
        Ok(())
    }
}
impl SketchCircularPlacementCheck {
    fn verify(
        &self,
        index: &SketchReadIndex<'_>,
        members: &[SketchEntityId],
        backend: &dyn SketchCircleConstruction,
    ) -> Result<(), SketchError> {
        let entity = indexed_entity(members, self.member)?;
        let seed = index.circular_seed(entity)?;
        backend
            .verify(self.points, seed, self.options)
            .map_err(|cause| {
                SketchError::Definition(format!(
                    "placement member {} verification: {cause}",
                    self.member
                ))
            })?;
        Ok(())
    }
}
