// SPDX-License-Identifier: GPL-2.0-only
//! Feature suppression through the document's transactional replay path.
use oblikovati_api::{Error, wire};
use oblikovati_martensite::{EvaluationFailure, PrismResolutionPolicy};
use oblikovati_model::{
    feature::{FeatureEvaluationStatus, FeatureId, part::PartFeatureFailure},
    history::DocumentSnapshot,
    identity::DocumentGuid,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    OwnedCheckpointCandidate, OwnedCheckpointReplay, TransactionObserver, files::FileWorkspace,
    workspace::WorkspaceObserver,
};
type ProjectedSuppression = (Vec<u8>, Option<OwnedCheckpointCandidate>);
type ReplayedSuppression = (
    FeatureEvaluationStatus<PartFeatureFailure<EvaluationFailure>>,
    OwnedCheckpointCandidate,
);

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn crate::ExtrudeRegionAccess>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: wire::SetFeatureSuppressedArgs =
        crate::decode(wire::METHOD_FEATURES_SET_SUPPRESSED, encoded)?;
    let backend = backend
        .ok_or_else(|| Error("feature suppression: expected a feature replay service".into()))?;
    let document = crate::part_owner::active_part(files)?;
    let id = FeatureId(request.id);
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    if let Some(replay) = backend
        .checkpoint_health_policy()
        .and_then(|policy| current_replay(files, document, policy))
    {
        return apply_checkpointed(files, backend, guid, request, replay);
    }
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Set feature suppression",
            |snapshot, _| set_suppression(snapshot, &request),
            |candidate, ()| {
                let health = backend.feature_health(candidate, guid, id).map_err(Error)?;
                detail(candidate, id, health)
            },
        )
        .map_err(api_error)
}

fn current_replay<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    policy: PrismResolutionPolicy,
) -> Option<OwnedCheckpointReplay> {
    let workspace = files.workspace();
    let session = workspace.session(document).ok()?;
    let policy = Some(policy);
    let prefix = session
        .owned_checkpoint_slot(policy)?
        .candidate_prefix(session)
        .ok()?;
    workspace
        .owned_checkpoint_replay(document, policy, prefix)
        .ok()
}

fn apply_checkpointed<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: &dyn crate::ExtrudeRegionAccess,
    guid: DocumentGuid,
    request: wire::SetFeatureSuppressedArgs,
    replay: OwnedCheckpointReplay,
) -> Result<Vec<u8>, Error> {
    let id = FeatureId(request.id);
    let ((reply, checkpoint), transition) = files
        .commands(replay.document())
        .map_err(api_error)?
        .run_projected_with_transition(
            "Set feature suppression",
            |snapshot, _| set_suppression(snapshot, &request),
            |snapshot, (), history| {
                let replayed = replay.prepare_feature_status(snapshot, history, id)?;
                project_suppression(snapshot, guid, id, backend, replayed)
            },
        )
        .map_err(api_error)?;
    if let Some(checkpoint) = checkpoint {
        crate::checkpoint_replay::adopt_session_candidate(files, &replay, checkpoint, &transition)
            .map_err(|error| {
                Error(format!(
                    "document {}: suppression committed but replay adoption failed: {error}",
                    replay.document()
                ))
            })?;
    }
    Ok(reply)
}

fn project_suppression(
    snapshot: &DocumentSnapshot,
    guid: DocumentGuid,
    id: FeatureId,
    backend: &dyn crate::ExtrudeRegionAccess,
    replayed: Option<ReplayedSuppression>,
) -> Result<ProjectedSuppression, Error> {
    let (health, checkpoint) = match replayed {
        Some((status, checkpoint)) => {
            let health = match status {
                FeatureEvaluationStatus::Healthy | FeatureEvaluationStatus::Suppressed => {
                    String::new()
                }
                status => format!("{status:?}"),
            };
            (health, Some(checkpoint))
        }
        None => (
            backend.feature_health(snapshot, guid, id).map_err(Error)?,
            None,
        ),
    };
    Ok((detail(snapshot, id, health)?, checkpoint))
}

fn set_suppression(
    snapshot: &mut DocumentSnapshot,
    request: &wire::SetFeatureSuppressedArgs,
) -> Result<(), Error> {
    snapshot
        .set_feature_suppressed(FeatureId(request.id), request.suppressed)
        .map_err(api_error)
}

fn detail(snapshot: &DocumentSnapshot, id: FeatureId, health: String) -> Result<Vec<u8>, Error> {
    let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
    let index = snapshot
        .work_geometry
        .features()
        .iter()
        .position(|item| item.id() == id)
        .ok_or_else(|| Error(format!("feature {id:?}: expected a history position")))?;
    crate::encode(&wire::FeatureDetailResult {
        feature: wire::FeatureDetail {
            feature_info: wire::FeatureInfo {
                id: id.0,
                name: feature.name().into(),
                kind: feature.definition().kind().into(),
                suppressed: snapshot.feature_suppressed(id).map_err(api_error)?,
                health,
            },
            index: i64::try_from(index).map_err(api_error)?,
            scalars: crate::feature_lifecycle::feature_scalars(feature, snapshot)?,
            slots: None,
        },
    })
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}
