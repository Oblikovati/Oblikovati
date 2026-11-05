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
    /// Optional start/end cap members: verify their centreline instead of the boundary arc.
    pub cap_members: Option<[usize; 2]>,
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
        let seed = self.solved_seed(index, members)?;
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
    fn solved_seed(
        &self,
        index: &SketchReadIndex<'_>,
        members: &[SketchEntityId],
    ) -> Result<crate::sketch::SolvedCircularSeed, SketchError> {
        let entity = indexed_entity(members, self.member)?;
        Ok(match self.cap_members {
            None => index.circular_seed(entity)?,
            Some([start, end]) => index.arc_slot_centerline(
                entity,
                [
                    indexed_entity(members, start)?,
                    indexed_entity(members, end)?,
                ],
            )?,
        })
    }
}
