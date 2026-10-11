// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentEditHistory, DocumentSnapshot, HistoryError, RecipeReservations};
use crate::{
    document_sketch::DocumentSketchId,
    identity::{RecipeIdAllocator, RecipeIdError},
    sketch::SketchEntityId,
};
use std::collections::BTreeSet;
/// Permanent sketch and entity namespace marks; never part of an undo snapshot.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct SketchReservationMarks {
    pub sketch: u64,
    pub entity: u64,
}
impl RecipeReservations<'_> {
    /// Reserve a sketch ID even if the edit later fails; example: `let id = ids.allocate_sketch()?;`.
    pub fn allocate_sketch(&mut self) -> Result<DocumentSketchId, RecipeIdError> {
        self.sketch_ids.allocate().map(DocumentSketchId)
    }
    /// Reserve a document-global sketch entity ID; example: `let point = ids.allocate_sketch_entity()?;`.
    pub fn allocate_sketch_entity(&mut self) -> Result<SketchEntityId, RecipeIdError> {
        self.sketch_entity_ids.allocate().map(SketchEntityId)
    }
}
impl DocumentEditHistory {
    /// Inspect permanent marks for save/copy; example: `history.sketch_reservations()`.
    pub fn sketch_reservations(&self) -> SketchReservationMarks {
        SketchReservationMarks {
            sketch: self.sketch_ids.high_water(),
            entity: self.sketch_entity_ids.high_water(),
        }
    }
    /// Restore saved marks without losing deleted reservations; example: `history.with_sketch_reservations(saved)?`.
    pub fn with_sketch_reservations(
        mut self,
        marks: SketchReservationMarks,
    ) -> Result<Self, RecipeIdError> {
        self.sketch_ids = restore_mark(
            marks.sketch,
            self.current.sketches.sketch_ids(),
            self.sketch_ids.high_water(),
        )?;
        self.sketch_entity_ids = restore_mark(
            marks.entity,
            self.current.sketches.entity_ids(),
            self.sketch_entity_ids.high_water(),
        )?;
        Ok(self)
    }
    pub(super) fn validate_sketch_ids(
        &self,
        candidate: &DocumentSnapshot,
        before: SketchReservationMarks,
    ) -> Result<(), HistoryError> {
        require_reserved(
            candidate.sketches.sketch_ids(),
            self.current.sketches.sketch_ids(),
            before.sketch,
            self.sketch_ids.high_water(),
            HistoryError::UnreservedSketchId,
        )?;
        require_reserved(
            candidate.sketches.entity_ids(),
            self.current.sketches.entity_ids(),
            before.entity,
            self.sketch_entity_ids.high_water(),
            HistoryError::UnreservedSketchEntityId,
        )
    }
}
pub(super) fn restore_mark(
    mark: u64,
    mut live: Vec<u64>,
    previous: u64,
) -> Result<RecipeIdAllocator, RecipeIdError> {
    if previous != 0 && !live.contains(&previous) {
        live.push(previous);
    }
    RecipeIdAllocator::restore(Some(mark), &live)
}
pub(super) fn require_reserved(
    candidate: Vec<u64>,
    current: Vec<u64>,
    before: u64,
    after: u64,
    failure: fn(u64) -> HistoryError,
) -> Result<(), HistoryError> {
    let current: BTreeSet<u64> = current.into_iter().collect();
    for id in candidate {
        if !current.contains(&id) && (id <= before || id > after) {
            return Err(failure(id));
        }
    }
    Ok(())
}
