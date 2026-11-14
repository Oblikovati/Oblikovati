// SPDX-License-Identifier: GPL-2.0-only
//! Shared part replay and freshness ownership for desktop and headless hosts.
use oblikovati_api::wire::FeatureError;
use oblikovati_martensite::{
    EvaluationFailure, NamedPrismEvaluator, NamedPrismSnapshot, PrismResolutionPolicy,
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

/// Part replay capability independent of viewport and topology queries.
pub trait PartUpdateAccess<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> {
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
    /// Replay a private candidate with the owner's policy; example: `owner.preview_datums(files, id, candidate)?`.
    fn preview_datums(
        &mut self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        candidate: &DocumentSnapshot,
    ) -> Result<oblikovati_model::feature::work_geometry::WorkGeometryEvaluation, String>;
}

/// A headless owner of generation-checked Martensite replay results.
#[derive(Default)]
pub struct PartReplayService {
    slots: BTreeMap<u64, PartEvaluationSlot<NamedPrismSnapshot, EvaluationFailure>>,
}
impl PartReplayService {
    fn current_slot<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<&PartEvaluationSlot<NamedPrismSnapshot, EvaluationFailure>, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
        if self
            .slots
            .get(&document)
            .is_none_or(|slot| slot.current(session).is_err())
        {
            self.update_part(files, document)?;
        }
        self.slots.get(&document).ok_or_else(|| {
            format!("document {document} part replay: expected an adopted current result")
        })
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> PartUpdateAccess<O, W, F>
    for PartReplayService
{
    fn update_part(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<FeatureError>, String> {
        self.slots
            .retain(|id, _| files.workspace().documents().contains(id));
        let completed = complete_part_replay(files, document, None)?;
        let slot = self.slots.entry(document).or_default();
        files
            .commands(document)
            .map_err(|error| error.to_string())?
            .adopt_part_evaluation(completed, slot)
            .map_err(|error| error.to_string())?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|error| error.to_string())?;
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
        Ok(part_replay_is_stale(session, self.slots.get(&document)))
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
        preview_part_datums(files, document, candidate, None)
    }
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
    let result = match resolution {
        Some(policy) => candidate.evaluate_part_features(
            &mut oblikovati_martensite::CertifiedNamedPrismEvaluator::with_policy(
                session.history(),
                guid,
                policy,
            ),
        ),
        None => {
            candidate.evaluate_part_features(&mut NamedPrismEvaluator::new(session.history(), guid))
        }
    }
    .map_err(|error| error.to_string())?;
    Ok(result.datum_evaluation())
}

/// Bind a named face through an adopted result; example: `resolve_slot_face(files, id, slot, &face)?`.
pub fn resolve_slot_face<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    slot: &PartEvaluationSlot<NamedPrismSnapshot, EvaluationFailure>,
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
        &NamedPrismEvaluator::new(session.history(), guid),
        face,
    )
}

/// Check whether a session's authored part needs a current replay.
/// Example: `part_replay_is_stale(session, slot)` before updating.
pub fn part_replay_is_stale<O: TransactionObserver>(
    session: &oblikovati_session::DocumentSession<O>,
    slot: Option<&PartEvaluationSlot<NamedPrismSnapshot, EvaluationFailure>>,
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
) -> Result<CompletedPartEvaluation<NamedPrismSnapshot, EvaluationFailure>, String> {
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(|error| error.to_string())?
        .internal_name;
    let session = workspace
        .session(document)
        .map_err(|error| error.to_string())?;
    let request = session.prepare_part_evaluation();
    match resolution {
        Some(policy) => request.evaluate(
            &mut oblikovati_martensite::CertifiedNamedPrismEvaluator::with_policy(
                session.history(),
                guid,
                policy,
            ),
        ),
        None => request.evaluate(&mut NamedPrismEvaluator::new(session.history(), guid)),
    }
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
