// SPDX-License-Identifier: GPL-2.0-only
//! Validate a private Hole candidate before committing its transaction (#4093).
use super::*;
use oblikovati_martensite::{CertifiedNamedSolidEvaluator, PrismResolutionPolicy};
use oblikovati_model::{
    history::{DocumentEditHistory, SnapshotReplay},
    identity::DocumentGuid,
};

pub(super) fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    guid: DocumentGuid,
    policy: PrismResolutionPolicy,
    prefix: crate::CertifiedPartPrefix,
    prepared: PreparedHole,
) -> Result<crate::fillet::BlendTransaction, Error> {
    if prefix.replay.policy() != Some(policy) || !prefix.replay.owns_identity(guid) {
        return Err(Error(format!(
            "hole document {document}: expected a current prefix with the same identity and resolution"
        )));
    }
    files.commands(document).map_err(api_error)?.run_projected_with_transition(
        "Create hole",
        |candidate, ids| prepared.create(candidate, ids),
        |candidate, feature, history| {
            let (bodies, captured) = validate_candidate(&prefix, history, candidate, feature, guid, policy)?;
            let authored = candidate.work_geometry.get(feature).map_err(api_error)?;
            let reply = crate::encode(&serde_json::json!({"id": feature.0, "name": authored.name(), "kind": authored.definition().kind(), "suppressed": candidate.feature_suppressed(feature).map_err(api_error)?, "health": "", "feature": authored.name(), "bodies": bodies, "healthy": true}))?;
            Ok((reply, captured))
        },
    ).map_err(api_error)
}
fn validate_candidate(
    prefix: &crate::CertifiedPartPrefix,
    history: &DocumentEditHistory,
    candidate: &DocumentSnapshot,
    feature: FeatureId,
    guid: DocumentGuid,
    policy: PrismResolutionPolicy,
) -> Result<(usize, Option<crate::CertifiedPartCandidate>), Error> {
    if let Some((count, candidate)) = prefix.replay.prepare(history, candidate, feature)? {
        return Ok((
            count,
            Some(crate::CertifiedPartCandidate { policy, candidate }),
        ));
    }
    let replay = SnapshotReplay::new(candidate).map_err(api_error)?;
    let mut evaluator = CertifiedNamedSolidEvaluator::with_policy(replay.history(), guid, policy);
    let result = replay
        .history()
        .snapshot()
        .evaluate_part_features(&mut evaluator)
        .map_err(api_error)?;
    if result.body_from_source(feature).is_err() {
        return Err(Error(format!(
            "hole {feature:?}, health {:?}: expected a valid named body",
            result.health()
        )));
    }
    Ok((result.ordered_bodies().count(), None))
}
