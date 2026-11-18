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
    TransactionObserver,
    files::FileWorkspace,
    part_evaluation::{CompletedPartEvaluation, PartEvaluationSlot},
    workspace::WorkspaceObserver,
};
use std::collections::BTreeMap;
mod access;
mod candidate;
pub use access::PartUpdateAccess;
pub use candidate::{
    CertifiedCheckpointCandidate, CertifiedCheckpointPrefix, CertifiedPartCandidate,
    CertifiedPartPrefix,
};
use candidate::{DocumentPartReplay, NamedSketchRecompute};

/// A headless owner of generation-checked Martensite replay results.
#[derive(Default)]
pub struct PartReplayService {
    slots: BTreeMap<u64, DocumentPartReplay>,
    resolution: Option<PrismResolutionPolicy>,
    replay_count: u64,
}
impl PartReplayService {
    /// Create a certified headless replay owner; example: `PartReplayService::with_policy(policy)?`.
    pub fn with_policy(policy: PrismResolutionPolicy) -> Result<Self, String> {
        Ok(Self {
            slots: BTreeMap::new(),
            resolution: Some(policy.validate()?),
            replay_count: 0,
        })
    }
    /// Count complete part replay attempts by this owner; example: `service.replay_count()`.
    pub fn replay_count(&self) -> u64 {
        self.replay_count
    }
    /// Borrow a document's retained replay; example: `service.slot(document)`.
    pub fn slot(
        &self,
        document: u64,
    ) -> Option<&PartEvaluationSlot<NamedSolid, EvaluationFailure>> {
        self.slots.get(&document).map(|owner| &owner.part)
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
    fn cached_checkpoint_prefix<'a>(
        &'a self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Option<CertifiedCheckpointPrefix<'a>>, String> {
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
        self.current_slot(files, document).map(|_| ())
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
        self.slot(document)
            .ok_or_else(|| format!("document {document}: expected a retained part replay"))?
            .current(session)
            .map_err(|error| error.to_string())
    }
    fn candidate_prefix<'a>(
        &'a self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<CertifiedPartPrefix<'a>, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        let prefix = self
            .slot(document)
            .ok_or_else(|| format!("document {document}: expected a retained part replay"))?
            .candidate_prefix(session)
            .map_err(|error| error.to_string())?;
        Ok(CertifiedPartPrefix {
            prefix,
            policy: self.resolution,
        })
    }
    fn blend_body(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        edges: &[Vec<u8>],
    ) -> Result<Vec<u8>, String> {
        let slot = self.current_slot(files, document)?;
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
        self.retain_open_documents(files.workspace().documents());
        let owner = self.slots.entry(document).or_default();
        refresh_and_adopt_counted(
            files,
            document,
            self.resolution,
            &mut owner.part,
            &mut self.replay_count,
        )?;
        owner.checkpoints = None;
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        part_feature_errors(
            session.snapshot(),
            owner
                .part
                .current(session)
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
        Ok(part_replay_is_stale(session, self.slot(document)))
    }
    fn resolve_planar_face(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        face: &oblikovati_model::feature::work_plane::WorkPlaneFaceReference,
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        let slot = self.current_slot(files, document)?;
        resolve_slot_face(files, document, slot, face)
    }
    fn datum_evaluation(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String> {
        let slot = self.current_slot(files, document)?;
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
        let slot = self.current_slot(files, document)?;
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
        let slot = self.current_slot(files, document)?;
        slot_datums_for_face(files, document, slot, face)
    }
}

/// Refresh sketches and publish a matching part result to the supplied owner.
/// Example: `refresh_and_adopt_part(files, id, policy, &mut slot)?`.
pub fn refresh_and_adopt_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    resolution: Option<PrismResolutionPolicy>,
    slot: &mut PartEvaluationSlot<NamedSolid, EvaluationFailure>,
) -> Result<(), String> {
    refresh_and_adopt_counted(files, document, resolution, slot, &mut 0)
}

fn refresh_and_adopt_counted<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    resolution: Option<PrismResolutionPolicy>,
    slot: &mut PartEvaluationSlot<NamedSolid, EvaluationFailure>,
    replay_count: &mut u64,
) -> Result<(), String> {
    let completed = prepare_part_sketches(files, document, resolution, replay_count)?;
    let published = if let Some(completed) = completed {
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_sketch_replay(completed, slot)
            .map_err(|error| error.to_string())?
    } else {
        false
    };
    if !published {
        let completed = complete_part_replay_counted(files, document, resolution, replay_count)?;
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_part_evaluation(completed, slot)
            .map_err(|error| error.to_string())?;
    }
    Ok(())
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
    let workspace = files.workspace();
    let session = workspace
        .session(document)
        .map_err(|error| error.to_string())?;
    let guid = workspace
        .identity(document)
        .map_err(|error| error.to_string())?
        .internal_name;
    let mut evaluator =
        CertifiedNamedSolidEvaluator::with_optional_policy(session.history(), guid, resolution);
    candidate
        .evaluate_part_features(&mut evaluator)
        .map(|result| result.datum_evaluation())
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

/// Replay against the current history with the selected tolerance policy.
/// Example: `complete_part_replay(files, id, None)?` before generation-checked adoption.
pub fn complete_part_replay<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    resolution: Option<PrismResolutionPolicy>,
) -> Result<CompletedPartEvaluation<NamedSolid, EvaluationFailure>, String> {
    complete_part_replay_counted(files, document, resolution, &mut 0)
}
fn complete_part_replay_counted<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    resolution: Option<PrismResolutionPolicy>,
    replay_count: &mut u64,
) -> Result<CompletedPartEvaluation<NamedSolid, EvaluationFailure>, String> {
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(|error| error.to_string())?
        .internal_name;
    let session = workspace
        .session(document)
        .map_err(|error| error.to_string())?;
    let request = session.prepare_part_evaluation();
    let mut evaluator =
        CertifiedNamedSolidEvaluator::with_optional_policy(session.history(), guid, resolution);
    *replay_count = replay_count.saturating_add(1);
    request
        .evaluate(&mut evaluator)
        .map_err(|error| error.to_string())
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
        DeleteBody(_) => "delete-body",
        Fillet(_) => "fillet",
        Chamfer(_) => "chamfer",
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
        AxisThroughPoints(_) => "two-points",
    }
}
