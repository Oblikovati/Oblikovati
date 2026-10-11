// SPDX-License-Identifier: GPL-2.0-only
use super::{DocumentSnapshot, sketch_projection::CurrentSketchProjectionInputs};
use crate::{
    document_sketch::{DocumentSketchId, DocumentSketchSolveOutcome, SketchSolveScope},
    sketch::SketchError,
};
use oblikovati_solve::{diagnostics::ConstraintStructure, nonlinear::SolveOptions};

impl DocumentSnapshot {
    /// Solve one sketch with current source coordinates; example: `snapshot.solve_one_document_sketch(id, options, || false)?`.
    pub fn solve_one_document_sketch(
        &self,
        id: DocumentSketchId,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<DocumentSketchSolveOutcome, SketchError> {
        self.sketches.get(id)?;
        Ok(self.solve_sketch_scope(options, cancelled, SketchSolveScope::One(id)))
    }
    /// Recompute all sketches from the current immutable source catalogue; example: `snapshot.solve_document_sketches(options, || false)`.
    pub fn solve_document_sketches(
        &self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> DocumentSketchSolveOutcome {
        self.solve_sketch_scope(options, cancelled, SketchSolveScope::All)
    }
    /// Recompute changed projection and dimension readers; example: `snapshot.solve_pending_document_sketches(options, || false)`.
    pub fn solve_pending_document_sketches(
        &self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> DocumentSketchSolveOutcome {
        self.solve_sketch_scope(options, cancelled, SketchSolveScope::Pending)
    }
    fn solve_sketch_scope(
        &self,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
        scope: SketchSolveScope,
    ) -> DocumentSketchSolveOutcome {
        let inputs = CurrentSketchProjectionInputs::new(self);
        self.sketches
            .solve_scope(&self.parameters, options, cancelled, scope, &inputs)
    }
    /// Inspect active structure from current reference inputs without a solve or mutation.
    /// Example: `snapshot.analyze_document_sketch(id, 1e-9)?`.
    pub fn analyze_document_sketch(
        &self,
        id: DocumentSketchId,
        rank_tolerance: f64,
    ) -> Result<ConstraintStructure, SketchError> {
        let placed = self.sketches.get(id)?;
        let anchors = self.resolve_sketch_projected_points(id)?;
        placed
            .geometry()
            .analyze_with_grounded_points(&self.parameters, &anchors, rank_tolerance)
    }
    /// Read current shapes and movement without solving or changing the document.
    /// Example: `snapshot.inspect_document_sketch(id, Default::default())?;`.
    pub fn inspect_document_sketch(
        &self,
        id: DocumentSketchId,
        options: crate::sketch::SketchMovementOptions,
    ) -> Result<Vec<crate::sketch::SketchEntityInspection>, SketchError> {
        let placed = self.sketches.get(id)?;
        let anchors = self.resolve_sketch_projected_points(id)?;
        placed
            .geometry()
            .inspect_entities(&self.parameters, &anchors, options)
    }
}
