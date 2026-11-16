// SPDX-License-Identifier: GPL-2.0-only
//! Resolve edit continuity against the published ordered part state.
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolutionPolicy};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentEditHistory, identity::DocumentGuid,
};
use oblikovati_session::SketchContinuityResolver;
use oblikovati_solve::nonlinear::SolveOptions;

/// A document-bound resolver with the same kernel and solver policies as its part owner.
pub struct PartSketchContinuity {
    document: DocumentGuid,
    resolution: Option<PrismResolutionPolicy>,
    options: SolveOptions,
}
impl PartSketchContinuity {
    /// Bind the document and policies; example: `PartSketchContinuity::new(guid, None, options)`.
    pub fn new(
        document: DocumentGuid,
        resolution: Option<PrismResolutionPolicy>,
        options: SolveOptions,
    ) -> Self {
        Self {
            document,
            resolution,
            options,
        }
    }
    /// Resolve a sketch plane from current ordered part data.
    /// Example: `resolver.plane(history, sketch)?` before opening the editor.
    pub fn plane(
        &self,
        history: &DocumentEditHistory,
        sketch: DocumentSketchId,
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        let snapshot = history.snapshot();
        let datums = self.datums(history)?;
        snapshot
            .resolve_sketch_plane_with_datums(sketch, &datums)
            .map_err(|error| error.to_string())
    }
    /// Replay the current history for read-only sketch inspection.
    /// Example: `resolver.datums(history)?` before reading sketch health.
    pub fn datums(
        &self,
        history: &DocumentEditHistory,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        let snapshot = history.snapshot();
        let mut evaluator = CertifiedNamedSolidEvaluator::with_optional_policy(
            history,
            self.document,
            self.resolution,
        );
        let (result, _) = snapshot
            .recompute_part_with_sketches(&mut evaluator, self.options)
            .map_err(|error| error.to_string())?;
        let datums = result.datum_evaluation();
        Ok(datums)
    }
}
impl SketchContinuityResolver for PartSketchContinuity {
    fn resolve(
        &self,
        history: &DocumentEditHistory,
        sketch: DocumentSketchId,
    ) -> Result<(), String> {
        self.plane(history, sketch).map(|_| ())
    }
}

#[cfg(test)]
#[path = "../tests/part_sketch_continuity/mod.rs"]
mod tests;
