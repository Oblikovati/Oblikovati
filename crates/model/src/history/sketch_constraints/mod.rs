// SPDX-License-Identifier: GPL-2.0-only
use crate::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    identity::{RecipeIdAllocator, RecipeIdError},
    sketch::SketchConstraintId,
};
use std::collections::BTreeMap;
mod error;
mod transition;
pub use error::SketchConstraintReservationError;
/// Permanent local constraint namespaces; retired sketches keep their allocation owners.
#[derive(Debug, Default)]
pub struct SketchConstraintReservations {
    owners: BTreeMap<u64, RecipeIdAllocator>,
}
impl SketchConstraintReservations {
    /// Restore explicit marks, or bootstrap an older record once from its live relations.
    /// Example: `SketchConstraintReservations::restore(&sketches, sketch_mark, Some(&marks))?`.
    pub fn restore(
        sketches: &DocumentSketches,
        sketch_mark: u64,
        marks: Option<&BTreeMap<u64, u64>>,
    ) -> Result<Self, RecipeIdError> {
        let mut restored = Self::default();
        restored.restore_explicit_owners(sketches, sketch_mark, marks)?;
        restored.restore_live_owners(sketches, sketch_mark, marks.is_some())?;
        Ok(restored)
    }
    fn restore_explicit_owners(
        &mut self,
        sketches: &DocumentSketches,
        sketch_mark: u64,
        marks: Option<&BTreeMap<u64, u64>>,
    ) -> Result<(), RecipeIdError> {
        let Some(marks) = marks else {
            return Ok(());
        };
        for (&sketch, &mark) in marks {
            require_sketch_namespace(sketch, sketch_mark)?;
            self.restore_owner(sketches, sketch, Some(mark))?;
        }
        Ok(())
    }
    fn restore_live_owners(
        &mut self,
        sketches: &DocumentSketches,
        sketch_mark: u64,
        explicit: bool,
    ) -> Result<(), RecipeIdError> {
        for sketch in sketches.sketches() {
            let id = sketch.id().0;
            require_sketch_namespace(id, sketch_mark)?;
            if !self.owners.contains_key(&id) {
                self.restore_owner(sketches, id, explicit.then_some(0))?;
            }
        }
        Ok(())
    }
    /// Inspect sorted nonzero marks for file capture; example: `let marks = reservations.marks();`.
    pub fn marks(&self) -> BTreeMap<u64, u64> {
        self.owners
            .iter()
            .filter_map(|(&sketch, owner)| {
                (owner.high_water() != 0).then_some((sketch, owner.high_water()))
            })
            .collect()
    }
    /// Allocate inside a reserved sketch namespace, even before a new sketch is published.
    /// Example: `reservations.allocate(sketch, sketch_mark)?` before adding the relation.
    pub fn allocate(
        &mut self,
        sketch: DocumentSketchId,
        sketch_mark: u64,
    ) -> Result<SketchConstraintId, SketchConstraintReservationError> {
        let failure = |cause| SketchConstraintReservationError { sketch, cause };
        require_sketch_namespace(sketch.0, sketch_mark).map_err(failure)?;
        self.owners
            .entry(sketch.0)
            .or_default()
            .allocate()
            .map(SketchConstraintId)
            .map_err(failure)
    }
    fn restore_owner(
        &mut self,
        sketches: &DocumentSketches,
        sketch: u64,
        mark: Option<u64>,
    ) -> Result<(), RecipeIdError> {
        let live = constraint_ids(sketches, DocumentSketchId(sketch));
        let owner = RecipeIdAllocator::restore(mark, &live)?;
        self.owners.insert(sketch, owner);
        Ok(())
    }
}
fn require_sketch_namespace(sketch: u64, mark: u64) -> Result<(), RecipeIdError> {
    if sketch == 0 {
        return Err(RecipeIdError::ZeroId);
    }
    if sketch > mark {
        return Err(RecipeIdError::BeyondHighWater {
            id: sketch,
            high_water: mark,
        });
    }
    Ok(())
}
fn constraint_ids(sketches: &DocumentSketches, sketch: DocumentSketchId) -> Vec<u64> {
    sketches
        .sketches()
        .iter()
        .find(|placed| placed.id() == sketch)
        .map(|placed| {
            placed
                .geometry()
                .constraints()
                .iter()
                .map(|constraint| constraint.id.0)
                .collect()
        })
        .unwrap_or_default()
}
