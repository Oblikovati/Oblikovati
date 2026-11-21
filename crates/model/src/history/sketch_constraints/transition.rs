// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::history::{
    DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations,
    sketch_ids::require_reserved,
};
impl RecipeReservations<'_> {
    /// Reserve a sketch-local constraint ID permanently; example: `let id = ids.allocate_sketch_constraint(sketch)?;`.
    pub fn allocate_sketch_constraint(
        &mut self,
        sketch: DocumentSketchId,
    ) -> Result<SketchConstraintId, SketchConstraintReservationError> {
        self.sketch_constraint_ids
            .allocate(sketch, self.sketch_ids.high_water())
    }
}
impl DocumentEditHistory {
    pub(in crate::history) fn validate_sketch_constraint_ids(
        &self,
        candidate: &DocumentSnapshot,
        before: &BTreeMap<u64, u64>,
    ) -> Result<(), HistoryError> {
        let after = self.sketch_constraint_ids.marks();
        for sketch in candidate.sketches.sketches() {
            let owner = sketch.id();
            require_reserved(
                constraint_ids(&candidate.sketches, owner),
                constraint_ids(&self.current.sketches, owner),
                before.get(&owner.0).copied().unwrap_or(0),
                after.get(&owner.0).copied().unwrap_or(0),
                |constraint| HistoryError::UnreservedSketchConstraintId {
                    sketch: owner.0,
                    constraint,
                },
            )?;
        }
        Ok(())
    }
    pub(in crate::history) fn restore_constraint_marks(
        &mut self,
        marks: &BTreeMap<u64, u64>,
    ) -> Result<(), RecipeIdError> {
        for (sketch, previous) in self.sketch_constraint_ids.marks() {
            let supplied = marks.get(&sketch).copied().unwrap_or(0);
            if supplied < previous {
                return Err(RecipeIdError::BeyondHighWater {
                    id: previous,
                    high_water: supplied,
                });
            }
        }
        self.sketch_constraint_ids = SketchConstraintReservations::restore(
            &self.current.sketches,
            self.sketch_ids.high_water(),
            Some(marks),
        )?;
        Ok(())
    }
}
