// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl DocumentPackage {
    /// Capture explicit permanent marks, including retired local constraint namespaces (#3776).
    /// Example: `package.with_sketch_reservations(&snapshot.sketches, history.sketch_reservations())?`.
    pub fn with_sketch_reservations(
        &self,
        sketches: &DocumentSketches,
        marks: SketchReservationMarks,
    ) -> Result<Self, PackageCodecError> {
        let previous = self.restore_sketches()?;
        require_monotonic_mark("sketch", marks.sketch, previous.sketch_ids.high_water())?;
        require_monotonic_mark("entity", marks.entity, previous.entity_ids.high_water())?;
        require_constraint_monotonicity(&previous.constraint_ids.marks(), &marks.constraints)?;
        let restored = restore_collection(
            sketches.clone(),
            marks.sketch,
            marks.entity,
            Some(&marks.constraints),
        )?;
        if restored.sketches.sketches().is_empty() && marks == SketchReservationMarks::default() {
            return Ok(self.clone());
        }
        self.capture_sketches(stored_reservations(restored, marks))
    }
    pub(super) fn initial_sketch_marks(
        &self,
        sketches: &DocumentSketches,
        sketch: u64,
        entity: u64,
    ) -> Result<SketchReservationMarks, PackageCodecError> {
        let mut constraints = self.restore_sketches()?.constraint_ids.marks();
        let live = SketchConstraintReservations::restore(sketches, sketch, None)
            .map_err(sketch_codec_error)?;
        for (owner, mark) in live.marks() {
            let retained = constraints.entry(owner).or_default();
            *retained = (*retained).max(mark);
        }
        Ok(SketchReservationMarks {
            sketch,
            entity,
            constraints,
        })
    }
}
fn require_constraint_monotonicity(
    retained: &BTreeMap<u64, u64>,
    supplied: &BTreeMap<u64, u64>,
) -> Result<(), PackageCodecError> {
    for (&sketch, &mark) in retained {
        require_monotonic_mark(
            &format!("sketch {sketch} constraint"),
            supplied.get(&sketch).copied().unwrap_or(0),
            mark,
        )?;
    }
    Ok(())
}

fn stored_reservations(
    restored: RestoredDocumentSketches,
    marks: SketchReservationMarks,
) -> StoredSketchCollection {
    StoredSketchCollection {
        recipe: restored.sketches.recipe(),
        sketch_high_water: marks.sketch,
        entity_high_water: marks.entity,
        constraint_high_water: Some(restored.constraint_ids.marks()),
    }
}
