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
        require_capture_marks(&previous, &marks)?;
        let restored = restore_collection(
            self,
            sketches.clone(),
            marks.sketch,
            marks.entity,
            Some(&marks.constraints),
            Some(&marks.dimension_names),
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
        let previous = self.restore_sketches()?;
        let mut constraints = previous.constraint_ids.marks();
        let mut dimension_names = previous.dimension_names.marks();
        let live = SketchConstraintReservations::restore(sketches, sketch, None)
            .map_err(sketch_codec_error)?;
        merge_local_marks(&mut constraints, live.marks());
        let live_names = restore_dimension_names(self, sketches, sketch, None)?;
        merge_local_marks(&mut dimension_names, live_names.marks());
        Ok(SketchReservationMarks {
            sketch,
            entity,
            constraints,
            dimension_names,
        })
    }
}
fn require_local_monotonicity(
    kind: &str,
    retained: &BTreeMap<u64, u64>,
    supplied: &BTreeMap<u64, u64>,
) -> Result<(), PackageCodecError> {
    for (&sketch, &mark) in retained {
        require_monotonic_mark(
            &format!("sketch {sketch} {kind}"),
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
        dimension_name_ordinals: Some(restored.dimension_names.marks()),
    }
}

fn require_capture_marks(
    previous: &RestoredDocumentSketches,
    marks: &SketchReservationMarks,
) -> Result<(), PackageCodecError> {
    require_monotonic_mark("sketch", marks.sketch, previous.sketch_ids.high_water())?;
    require_monotonic_mark("entity", marks.entity, previous.entity_ids.high_water())?;
    require_local_monotonicity(
        "constraint",
        &previous.constraint_ids.marks(),
        &marks.constraints,
    )?;
    require_local_monotonicity(
        "dimension name",
        &previous.dimension_names.marks(),
        &marks.dimension_names,
    )?;
    Ok(())
}

fn merge_local_marks(retained: &mut BTreeMap<u64, u64>, live: BTreeMap<u64, u64>) {
    for (owner, mark) in live {
        let prior = retained.entry(owner).or_default();
        *prior = (*prior).max(mark);
    }
}
