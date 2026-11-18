// SPDX-License-Identifier: GPL-2.0-only
//! Reuse document-owned results while validating an Extrude transaction.
use super::*;

pub(super) struct ExtrudeProjection<'a> {
    pub(super) backend: &'a dyn ExtrudeRegionAccess,
    pub(super) guid: DocumentGuid,
    pub(super) name: &'a str,
    pub(super) prefix: Option<&'a PolicyExtrudePrefix>,
}
impl ExtrudeProjection<'_> {
    pub(super) fn project(
        &self,
        candidate: &DocumentSnapshot,
        id: FeatureId,
        history: &oblikovati_model::history::DocumentEditHistory,
    ) -> Result<(Vec<u8>, Option<CapturedExtrudeCandidate>), Error> {
        let replay = self.capture(candidate, id, history)?;
        let (bodies, captured) = match replay {
            Some((count, captured)) => (count, Some(captured)),
            None => (
                self.backend
                    .validate(candidate, self.guid, id)
                    .map_err(Error)?,
                None,
            ),
        };
        Ok((
            extrude_response(candidate, id, self.name, bodies)?,
            captured,
        ))
    }
    fn capture(
        &self,
        candidate: &DocumentSnapshot,
        id: FeatureId,
        history: &oblikovati_model::history::DocumentEditHistory,
    ) -> Result<Option<ExtrudePrefixReplay>, Error> {
        let Some((_, prefix)) = self.prefix else {
            return Ok(None);
        };
        if candidate.sketches.requires_part_dependency_recompute() {
            return Ok(None);
        }
        self.backend
            .capture_with_prefix(history, self.guid, candidate, id, prefix)
            .map_err(Error)
    }
}
fn extrude_response(
    candidate: &DocumentSnapshot,
    id: FeatureId,
    name: &str,
    bodies: usize,
) -> Result<Vec<u8>, Error> {
    let feature = candidate.work_geometry.get(id).map_err(api_error)?;
    crate::encode(&serde_json::json!({
        "id": id.0, "name": feature.name(), "kind": feature.definition().kind(),
        "suppressed": candidate.feature_suppressed(id).map_err(api_error)?,
        "health": "", "feature": name, "bodies": bodies, "healthy": true
    }))
}
pub(super) fn prepare_owned_prefix<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    guid: DocumentGuid,
    backend: &dyn ExtrudeRegionAccess,
) -> Result<Option<PolicyExtrudePrefix>, Error> {
    let Some(policy) = backend.replay_policy() else {
        return Ok(None);
    };
    let session = files.workspace().session(document).map_err(api_error)?;
    if session.in_transaction() {
        return Ok(None);
    }
    if let Some(prefix) = session.owned_part_prefix(Some(policy)) {
        return Ok(Some((policy, prefix)));
    }
    if !session.snapshot().work_geometry.features().is_empty() {
        return Ok(None);
    }
    let mut evaluator = oblikovati_martensite::CertifiedNamedSolidEvaluator::with_policy(
        session.history(),
        guid,
        policy,
    );
    let completed = session
        .prepare_part_evaluation()
        .evaluate(&mut evaluator)
        .map_err(api_error)?;
    files
        .commands(document)
        .map_err(api_error)?
        .adopt_owned_part(Some(policy), completed)
        .map_err(Error)?;
    Ok(files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .owned_part_prefix(Some(policy))
        .map(|prefix| (policy, prefix)))
}
pub(super) fn adopt_extrude_candidate<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    guid: DocumentGuid,
    policy: oblikovati_martensite::PrismResolutionPolicy,
    candidate: CapturedExtrudeCandidate,
    transition: &oblikovati_model::history::EvaluationTransition,
) -> Result<(), Error> {
    let session = files.workspace().session(document).map_err(api_error)?;
    let completed = oblikovati_martensite::complete_named_candidate(
        candidate.into_transfer(),
        document,
        transition,
        session.history(),
        guid,
    )
    .map_err(Error)?;
    let completed = session.complete_part_transfer(completed).map_err(Error)?;
    files
        .commands(document)
        .map_err(api_error)?
        .adopt_owned_part(Some(policy), completed)
        .map_err(|error| {
            Error(format!(
                "extrusion committed for document {document}, but replay adoption failed: {error}"
            ))
        })
}
