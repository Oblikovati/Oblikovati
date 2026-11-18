// SPDX-License-Identifier: GPL-2.0-only
//! Preserve clean part checkpoints across a parameter value transaction.
use crate::{command_owner::CommandOwner, edit::PreparedEdit};
use oblikovati_api::Error;
use oblikovati_martensite::{
    CapturedNamedSolid, CertifiedNamedSolidEvaluator, EvaluationFailure, NamedSolid,
    PrismResolutionPolicy,
};
use oblikovati_model::{
    history::{DocumentEditHistory, DocumentSnapshot, EvaluationTransition},
    identity::DocumentGuid,
};
use oblikovati_session::part_evaluation::{CapturedPartCheckpointCandidate, PartCheckpointPrefix};

type NamedPrefix = PartCheckpointPrefix<NamedSolid, EvaluationFailure>;
type NamedCandidate = CapturedPartCheckpointCandidate<CapturedNamedSolid, EvaluationFailure>;
struct ParameterReplay {
    document: u64,
    guid: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
    prefix: NamedPrefix,
}
impl ParameterReplay {
    fn prepare(
        &self,
        snapshot: &DocumentSnapshot,
        history: &DocumentEditHistory,
    ) -> Result<Option<NamedCandidate>, Error> {
        if snapshot.sketches.requires_part_dependency_recompute() {
            return Ok(None);
        }
        let Some(change) = history.snapshot().part_replay_change(snapshot) else {
            return Ok(None);
        };
        if change.start == 0 {
            return Ok(None);
        }
        let mut evaluator =
            CertifiedNamedSolidEvaluator::with_optional_policy(history, self.guid, self.policy);
        let Some(candidate) = self
            .prefix
            .prepare_checkpoints(history, snapshot, &mut evaluator)
            .map_err(Error)?
        else {
            return Ok(None);
        };
        candidate
            .capture(history, |_, body| body.capture_detached(history, self.guid))
            .map(Some)
            .map_err(Error)
    }
    fn adopt(
        &self,
        owner: &mut impl CommandOwner,
        candidate: NamedCandidate,
        transition: &EvaluationTransition,
    ) -> Result<(), Error> {
        let session = owner
            .workspace()
            .session(self.document)
            .map_err(|error| Error(error.to_string()))?;
        let completed = oblikovati_martensite::complete_named_checkpoints(
            candidate.into_transfer(),
            self.document,
            transition,
            session.history(),
            self.guid,
        )
        .map_err(Error)?;
        let completed = session
            .complete_checkpoint_transfer(completed)
            .map_err(Error)?;
        owner
            .commands(self.document)
            .map_err(|error| Error(error.to_string()))?
            .adopt_owned_replayed_checkpoints(self.policy, completed)
            .map_err(Error)
    }
}
fn current_prefix(owner: &impl CommandOwner, document: u64) -> Option<ParameterReplay> {
    let workspace = owner.workspace();
    let session = workspace.session(document).ok()?;
    if session.in_transaction() {
        return None;
    }
    let (policy, prefix) = session.owned_checkpoint_prefix()?;
    let guid = workspace.identity(document).ok()?.internal_name;
    Some(ParameterReplay {
        document,
        guid,
        policy,
        prefix,
    })
}
/// Run a parameter edit with a current checkpoint when its prefix is safe.
/// Example: `try_edit(owner, edit, document)?` before a normal edit.
pub(super) fn try_edit(
    owner: &mut impl CommandOwner,
    edit: PreparedEdit,
    document: u64,
) -> Result<Vec<u8>, Error> {
    let Some(replay) = current_prefix(owner, document) else {
        return crate::command_owner::apply_edit(owner, edit, document);
    };
    let label = edit.label();
    let ((response, candidate), transition) = owner
        .commands(document)
        .map_err(|error| Error(error.to_string()))?
        .run_projected_with_transition(
            label,
            |snapshot, reservations| edit.apply(snapshot, reservations),
            |snapshot, response, history| {
                Ok::<_, Error>((response, replay.prepare(snapshot, history)?))
            },
        )
        .map_err(|error| Error(error.to_string()))?;
    if let Some(candidate) = candidate {
        replay.adopt(owner, candidate, &transition).map_err(|error| {
            Error(format!("document {document}: parameter edit committed but replay adoption failed: {error}"))
        })?;
    }
    owner.refresh_after_command(document)?;
    Ok(response)
}
