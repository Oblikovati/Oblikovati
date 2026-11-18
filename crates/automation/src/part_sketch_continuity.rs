// SPDX-License-Identifier: GPL-2.0-only
//! Resolve edit continuity against the published ordered part state.
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolutionPolicy};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentEditHistory, identity::DocumentGuid,
};
use oblikovati_session::{
    SketchContinuityResolver, TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
use oblikovati_solve::nonlinear::SolveOptions;

/// A document-bound resolver with the same kernel and solver policies as its part owner.
pub struct PartSketchContinuity {
    document: DocumentGuid,
    resolution: Option<PrismResolutionPolicy>,
    options: SolveOptions,
    #[cfg(test)]
    replay_count: std::sync::atomic::AtomicUsize,
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
            #[cfg(test)]
            replay_count: std::sync::atomic::AtomicUsize::new(0),
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
    /// Read matching current datums without replay; example: `resolver.cached_datums(workspace, id)?`.
    pub fn cached_datums<O: TransactionObserver, W: WorkspaceObserver>(
        &self,
        workspace: &WorkspaceSession<O, W>,
        document: u64,
    ) -> Result<
        Option<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation>,
        oblikovati_api::Error,
    > {
        let guid = workspace
            .identity(document)
            .map_err(|error| oblikovati_api::Error(error.to_string()))?
            .internal_name;
        if guid != self.document {
            return Err(oblikovati_api::Error(format!(
                "document {document} identity {guid:?}: expected sketch continuity owner {:?}",
                self.document
            )));
        }
        let session = workspace
            .session(document)
            .map_err(|error| oblikovati_api::Error(error.to_string()))?;
        if self.options != session.sketch_solve_options() {
            return Ok(None);
        }
        Ok(session
            .owned_part_slot(self.resolution)
            .and_then(|slot| slot.current(session).ok())
            .map(|part| part.datum_evaluation()))
    }
    /// Read current sketch datums with conservative replay when no matching result exists.
    /// Example: `resolver.datums_for_document(workspace, id)?` for a sketch query.
    pub fn datums_for_document<O: TransactionObserver, W: WorkspaceObserver>(
        &self,
        workspace: &WorkspaceSession<O, W>,
        document: u64,
    ) -> Result<
        oblikovati_model::feature::work_geometry::WorkGeometryEvaluation,
        oblikovati_api::Error,
    > {
        if let Some(datums) = self.cached_datums(workspace, document)? {
            return Ok(datums);
        }
        let session = workspace
            .session(document)
            .map_err(|error| oblikovati_api::Error(error.to_string()))?;
        Self::new(
            self.document,
            self.resolution,
            session.sketch_solve_options(),
        )
        .datums(session.history())
        .map_err(oblikovati_api::Error)
    }
    /// Replay the current history for read-only sketch inspection.
    /// Example: `resolver.datums(history)?` before reading sketch health.
    pub fn datums(
        &self,
        history: &DocumentEditHistory,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        #[cfg(test)]
        self.replay_count
            .fetch_add(1, std::sync::atomic::Ordering::Relaxed);
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
        sketches: &[DocumentSketchId],
    ) -> Vec<Result<(), String>> {
        let datums = self.datums(history);
        sketches
            .iter()
            .map(|sketch| match &datums {
                Ok(datums) => history
                    .snapshot()
                    .resolve_sketch_plane_with_datums(*sketch, datums)
                    .map(|_| ())
                    .map_err(|error| error.to_string()),
                Err(error) => Err(error.clone()),
            })
            .collect()
    }
}

#[cfg(test)]
#[path = "../tests/part_sketch_continuity/mod.rs"]
mod tests;
