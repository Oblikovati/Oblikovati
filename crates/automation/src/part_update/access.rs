// SPDX-License-Identifier: GPL-2.0-only
//! Generation-checked part replay access.
use super::*;

/// Part replay capability independent of viewport and topology queries.
pub trait PartUpdateAccess<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> {
    /// Borrow a current policy-scoped checkpoint when one is retained.
    /// Example: `owner.cached_checkpoint_prefix(files, id)?` before an edit.
    fn cached_checkpoint_prefix<'a>(
        &'a self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Option<CertifiedCheckpointPrefix<'a>>, String>;
    /// Publish an accepted policy-matched candidate; example: `owner.adopt_candidate(files, id, candidate, &transition)?`.
    fn adopt_candidate(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        candidate: CertifiedPartCandidate,
        transition: &oblikovati_model::history::EvaluationTransition,
    ) -> Result<(), String>;
    /// Report the policy used for retained results; example: `owner.resolution_policy()`.
    fn resolution_policy(&self) -> Option<PrismResolutionPolicy>;
    /// Ensure a current result exists; example: `owner.ensure_part(files, id)?` before a query.
    fn ensure_part(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String>;
    /// Borrow the current generation only; example: `owner.current_part(files, id)?`.
    fn current_part<'a>(
        &'a self,
        files: &'a FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<
        &'a oblikovati_model::feature::part::PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
        String,
    >;
    /// Capture a current candidate prefix; example: `owner.candidate_prefix(files, id)?`.
    fn candidate_prefix<'a>(
        &'a self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<CertifiedPartPrefix<'a>, String>;
    /// Resolve edge keys against the current part; example: `owner.blend_body(files, id, &edges)?`.
    fn blend_body(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String>;
    /// Replay a part and report failures; example: `owner.update_part(files, id)?`.
    fn update_part(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<FeatureError>, String>;
    /// Check the current replay generation; example: `owner.part_requires_update(files, id)?`.
    fn part_requires_update(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<bool, String>;
    /// Resolve a named face in the current part; example: `owner.resolve_planar_face(files, id, &face)?`.
    fn resolve_planar_face(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<oblikovati_model::sketch::SketchPlane, String>;
    /// Return current datum geometry and health; example: `owner.datum_evaluation(files, id)?`.
    fn datum_evaluation(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String>;
    /// Seal the current part datums to their history generation.
    /// Example: `owner.current_part_datums(files, id)?` before sketch creation.
    fn current_part_datums(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_session::part_evaluation::CurrentPartDatums, String>;
    /// Seal current part inputs for a new sketch on a named face.
    /// Example: `owner.current_part_datums_for_face(files, id, &face)?`.
    fn current_part_datums_for_face(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<oblikovati_session::part_evaluation::CurrentPartDatums, String>;
    /// Replay a private candidate with the owner's policy; example: `owner.preview_datums(files, id, candidate)?`.
    fn preview_datums(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        candidate: &DocumentSnapshot,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String>;
}
