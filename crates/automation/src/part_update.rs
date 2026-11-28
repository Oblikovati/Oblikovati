// SPDX-License-Identifier: GPL-2.0-only
//! Shared part replay and freshness ownership for desktop and headless hosts.
use oblikovati_api::wire::FeatureError;
use oblikovati_martensite::{
    CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid, PrismResolutionPolicy,
};
use oblikovati_model::{
    feature::{
        FeatureEvaluationStatus, FeatureHealthRecord,
        part::{PartFeatureDefinition, PartFeatureFailure},
    },
    history::DocumentSnapshot,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    TransactionObserver, files::FileWorkspace, part_evaluation::PartEvaluationSlot,
    workspace::WorkspaceObserver,
};
mod access;
mod candidate;
pub use access::PartUpdateAccess;
use candidate::NamedSketchRecompute;
pub use candidate::{
    CertifiedCheckpointCandidate, CertifiedCheckpointPrefix, CertifiedPartCandidate,
    CertifiedPartPrefix,
};

/// A headless owner of generation-checked Martensite replay results.
#[derive(Default)]
pub struct PartReplayService {
    resolution: Option<PrismResolutionPolicy>,
    replay_count: u64,
}
impl PartReplayService {
    /// Create a certified headless replay owner; example: `PartReplayService::with_policy(policy)?`.
    pub fn with_policy(policy: PrismResolutionPolicy) -> Result<Self, String> {
        Ok(Self {
            resolution: Some(policy.validate()?),
            replay_count: 0,
        })
    }
    /// Count complete part replay attempts by this owner; example: `service.replay_count()`.
    pub fn replay_count(&self) -> u64 {
        self.replay_count
    }
    /// Borrow a document-owned replay under this service's policy.
    /// Example: `service.slot(files, document)` before a query.
    pub fn slot<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &'a FileWorkspace<O, W, F>,
        document: u64,
    ) -> Option<&'a PartEvaluationSlot<NamedSolid, EvaluationFailure>> {
        files
            .workspace()
            .session(document)
            .ok()?
            .owned_part_slot(self.resolution)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> PartUpdateAccess<O, W, F>
    for PartReplayService
{
    fn adopt_checkpoints(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        candidate: CertifiedCheckpointCandidate,
        transition: &oblikovati_model::history::EvaluationTransition,
    ) -> Result<(), String> {
        self.adopt_named_checkpoints(files, document, candidate, transition)
    }
    fn cached_checkpoint_prefix(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Option<CertifiedCheckpointPrefix>, String> {
        Ok(self
            .checkpoint_prefix(files, document)?
            .map(|prefix| CertifiedCheckpointPrefix {
                prefix,
                policy: self.resolution,
            }))
    }
    fn adopt_candidate(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        candidate: CertifiedPartCandidate,
        transition: &oblikovati_model::history::EvaluationTransition,
    ) -> Result<(), String> {
        self.adopt_named_candidate(files, document, candidate, transition)
    }

    fn resolution_policy(&self) -> Option<PrismResolutionPolicy> {
        self.resolution
    }
    fn ensure_part(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        self.ensure_current(files, document)
    }
    fn current_part<'a>(
        &'a self,
        files: &'a FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<
        &'a oblikovati_model::feature::part::PartFeatureEvaluation<NamedSolid, EvaluationFailure>,
        String,
    > {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        self.slot(files, document)
            .ok_or_else(|| format!("document {document}: expected a retained part replay"))?
            .current(session)
            .map_err(|error| error.to_string())
    }
    fn candidate_prefix(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<CertifiedPartPrefix, String> {
        let replay = files
            .workspace()
            .owned_part_replay(document, self.resolution)
            .map_err(|error| error.to_string())?;
        Ok(CertifiedPartPrefix { replay })
    }
    fn blend_body(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        self.ensure_current(files, document)?;
        let slot = self
            .slot(files, document)
            .ok_or("expected document-owned replay")?;
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|error| error.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let evaluation = slot.current(session).map_err(|error| error.to_string())?;
        crate::fillet::resolve_blend_body(evaluation, session.history(), guid, edges)
    }
    fn update_part(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<FeatureError>, String> {
        files.recompute_owned_part(document, self.resolution, &mut self.replay_count)?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let slot = self
            .slot(files, document)
            .ok_or("expected document-owned replay")?;
        part_feature_errors(
            session.snapshot(),
            slot.current(session)
                .map_err(|error| error.to_string())?
                .health(),
        )
    }
    fn part_requires_update(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<bool, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        Ok(part_replay_is_stale(session, self.slot(files, document)))
    }
    fn resolve_planar_face(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        self.ensure_current(files, document)?;
        let slot = self
            .slot(files, document)
            .ok_or("expected document-owned replay")?;
        resolve_slot_face(files, document, slot, face)
    }
    fn datum_evaluation(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        self.ensure_current(files, document)?;
        let slot = self
            .slot(files, document)
            .ok_or("expected document-owned replay")?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        Ok(slot
            .current(session)
            .map_err(|error| error.to_string())?
            .datum_evaluation())
    }
    fn preview_datums(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        candidate: &DocumentSnapshot,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        preview_part_datums(files, document, candidate, self.resolution)
    }
    fn current_part_datums(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_session::part_evaluation::CurrentPartDatums, String> {
        self.ensure_current(files, document)?;
        let slot = self
            .slot(files, document)
            .ok_or("expected document-owned replay")?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        slot.current_datums(session)
            .map_err(|error| error.to_string())
    }
    fn current_part_datums_for_face(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<oblikovati_session::part_evaluation::CurrentPartDatums, String> {
        self.ensure_current(files, document)?;
        let slot = self
            .slot(files, document)
            .ok_or("expected document-owned replay")?;
        slot_datums_for_face(files, document, slot, face)
    }
}

/// Refresh dependent sketches from ordered replay before publishing a part.
/// Example: `refresh_part_sketches(files, id, None)?` on the part owner thread.
pub fn refresh_part_sketches<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    resolution: Option<PrismResolutionPolicy>,
) -> Result<bool, String> {
    let Some(completed) = prepare_part_sketches(files, document, resolution, &mut 0)? else {
        return Ok(false);
    };
    files
        .commands(document)
        .map_err(|error| error.to_string())?
        .adopt_recomputed_part_sketches(completed)
        .map_err(|error| error.to_string())
}
fn prepare_part_sketches<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    resolution: Option<PrismResolutionPolicy>,
    replay_count: &mut u64,
) -> Result<Option<NamedSketchRecompute>, String> {
    let workspace = files.workspace();
    let session = workspace
        .session(document)
        .map_err(|error| error.to_string())?;
    if !crate::datums::requires_part_replay(session.snapshot()) {
        return Ok(None);
    }
    let options = session.sketch_solve_options();
    let guid = workspace
        .identity(document)
        .map_err(|error| error.to_string())?
        .internal_name;
    let request = session.prepare_part_evaluation();
    let mut evaluator =
        CertifiedNamedSolidEvaluator::with_optional_policy(session.history(), guid, resolution);
    *replay_count = replay_count.saturating_add(1);
    let completed = request
        .recompute_sketches(&mut evaluator, options)
        .map_err(|error| error.to_string())?;
    Ok(Some(completed))
}

/// Evaluate a private candidate; example: `preview_part_datums(files, id, candidate, None)?`.
pub fn preview_part_datums<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    candidate: &DocumentSnapshot,
    resolution: Option<PrismResolutionPolicy>,
) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
    files
        .workspace()
        .preview_owned_part(document, resolution, candidate)
        .map(|preview| preview.evaluation.datum_evaluation())
        .map_err(|error| error.to_string())
}

/// Bind a named face through an adopted result; example: `resolve_slot_face(files, id, slot, &face)?`.
pub fn resolve_slot_face<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    slot: &PartEvaluationSlot<NamedSolid, EvaluationFailure>,
    face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
) -> Result<oblikovati_model::sketch::SketchPlane, String> {
    let workspace = files.workspace();
    let session = workspace
        .session(document)
        .map_err(|error| error.to_string())?;
    let guid = workspace
        .identity(document)
        .map_err(|error| error.to_string())?
        .internal_name;
    slot.resolve_planar_face(
        session,
        &CertifiedNamedSolidEvaluator::with_optional_policy(session.history(), guid, None),
        face,
    )
}

/// Seal a prospective support from the same adopted replay.
/// Example: `slot_datums_for_face(files, document, slot, &face)?`.
pub fn slot_datums_for_face<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    slot: &PartEvaluationSlot<NamedSolid, EvaluationFailure>,
    face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
) -> Result<oblikovati_session::part_evaluation::CurrentPartDatums, String> {
    let workspace = files.workspace();
    let session = workspace
        .session(document)
        .map_err(|error| error.to_string())?;
    let guid = workspace
        .identity(document)
        .map_err(|error| error.to_string())?
        .internal_name;
    slot.current_datums_for_face(
        session,
        &CertifiedNamedSolidEvaluator::with_optional_policy(session.history(), guid, None),
        face,
    )
}

/// Check whether a session's authored part needs a current replay.
/// Example: `part_replay_is_stale(session, slot)` before updating.
pub fn part_replay_is_stale<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
    slot: Option<&PartEvaluationSlot<NamedSolid, EvaluationFailure>>,
) -> bool {
    !session.snapshot().work_geometry.features().is_empty()
        && slot.is_none_or(|result| result.current(session).is_err())
}

/// Convert ordered replay health to canonical feature errors.
/// Example: `part_feature_errors(snapshot, evaluation.health())?` after replay.
pub fn part_feature_errors(
    snapshot: &DocumentSnapshot,
    health: &[FeatureHealthRecord<PartFeatureFailure<EvaluationFailure>>],
) -> Result<Vec<FeatureError>, String> {
    health
        .iter()
        .filter_map(|record| feature_error(snapshot, record).transpose())
        .collect()
}
fn feature_error(
    snapshot: &DocumentSnapshot,
    record: &FeatureHealthRecord<PartFeatureFailure<EvaluationFailure>>,
) -> Result<Option<FeatureError>, String> {
    let message = match &record.status {
        FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed => return Ok(None),
        status => format!("{status:?}"),
    };
    let feature = snapshot
        .work_geometry
        .get(record.feature)
        .map_err(|error| error.to_string())?;
    Ok(Some(FeatureError {
        id: record.feature.0,
        name: feature.name().into(),
        kind: feature_kind(feature.definition()).into(),
        message,
    }))
}
fn feature_kind(definition: &PartFeatureDefinition) -> &'static str {
    use PartFeatureDefinition::*;
    match definition {
        SketchPrism(_) => "extrude",
        SketchSweep(_) => "sweep",
        SketchLoft(_) => "loft",
        SketchRevolve(_) => "revolve",
        SketchCoil(_) => "coil",
        SketchEmboss(_) => "emboss",
        SketchRib(_) => "rib",
        SketchDecal(_) => "decal",
        DerivedPart(_) => "derive",
        ImportedModel(_) => "import",
        Unwrap(_) => "unwrap",
        DeleteBody(_) => "delete-body",
        Fillet(_) => "fillet",
        Chamfer(_) => "chamfer",
        Hole(_) => "hole",
        Slot(_) => "slot",
        Combine(_) => "combine",
        DirectEdit(_) => "direct-edit",
        Draft(_) => "draft",
        Thread(_) => "thread",
        DeleteFace(_) => "delete-face",
        ThickenOffset(_) => "thicken-offset",
        Split(_) => "split",
        Mark(_) => "mark",
        Shell(_) => "shell",
        Mirror(_) => "mirror",
        CircularPattern(_) => "circular-pattern",
        RectangularPattern(_) => "rectangular-pattern",
        MoveBody(_) => "move-body",
        Fixed(_) => "fixed-frame",
        Offset { .. } => "plane-offset",
        ParallelThroughPoint(_) => "plane-point",
        ThroughThreePoints(_) => "three-points",
        LineThroughPoint(_) => "line-point",
        NormalToAxis(_) => "normal-to-curve",
        AtAngleAboutAxis(_) => "line-plane-angle",
        ThroughTwoAxes(_) => "two-lines",
        Axis(_) => "line",
        Point(_) => "position",
        PointAtReference(_) => "point-reference",
        WorkPoint(_) => "work-point",
        AxisThroughPoints(_) => "two-points",
    }
}

#[cfg(test)]
#[path = "part_update/freshness_tests.rs"]
mod freshness_tests;
