// SPDX-License-Identifier: GPL-2.0-only
//! Preserve clean part checkpoints across an accepted document edit.
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
pub(super) type NamedCandidate =
    CapturedPartCheckpointCandidate<CapturedNamedSolid, EvaluationFailure>;
pub(super) struct CheckpointReplay {
    document: u64,
    guid: DocumentGuid,
    policy: Option<PrismResolutionPolicy>,
    prefix: NamedPrefix,
}
impl CheckpointReplay {
    pub(super) fn prepare(
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
    pub(super) fn adopt(
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
fn current_prefix(owner: &impl CommandOwner, document: u64) -> Option<CheckpointReplay> {
    let workspace = owner.workspace();
    let session = workspace.session(document).ok()?;
    if session.in_transaction() {
        return None;
    }
    let (policy, prefix) = session.owned_checkpoint_prefix()?;
    replay_for_prefix(workspace, document, policy, prefix)
}
pub(super) fn certified_prefix(
    owner: &impl CommandOwner,
    document: u64,
    prefix: crate::CertifiedCheckpointPrefix,
) -> Option<CheckpointReplay> {
    let workspace = owner.workspace();
    let session = workspace.session(document).ok()?;
    if session.in_transaction() {
        return None;
    }
    replay_for_prefix(workspace, document, prefix.policy, prefix.prefix)
}

/// Commit an edit and publish a matching checkpoint when its prefix is safe.
/// Example: `edit_with_owner(files, owner, document, label, edit)?`.
pub(super) fn edit_with_owner<
    R,
    O: oblikovati_session::TransactionObserver,
    W: oblikovati_session::workspace::WorkspaceObserver,
    F: oblikovati_persistence::file_access::DocumentFiles,
>(
    files: &mut oblikovati_session::files::FileWorkspace<O, W, F>,
    owner: &mut dyn crate::PartTopologyAccess<O, W, F>,
    document: u64,
    label: &str,
    edit: impl FnOnce(
        &mut DocumentSnapshot,
        &mut oblikovati_model::history::RecipeReservations<'_>,
    ) -> Result<R, Error>,
) -> Result<(R, bool), Error> {
    let policy = owner.resolution_policy();
    let prefix = owner
        .cached_checkpoint_prefix(files, document)
        .map_err(Error)?;
    let replay = prefix
        .filter(|prefix| prefix.policy == policy)
        .and_then(|prefix| certified_prefix(files, document, prefix));
    let Some((policy, replay)) = policy.zip(replay) else {
        return files
            .commands(document)
            .map_err(|error| Error(error.to_string()))?
            .run(label, edit)
            .map(|reply| (reply, false))
            .map_err(|error| Error(error.to_string()));
    };
    apply_checkpointed_edit(files, owner, document, label, edit, policy, replay)
}

fn apply_checkpointed_edit<
    R,
    O: oblikovati_session::TransactionObserver,
    W: oblikovati_session::workspace::WorkspaceObserver,
    F: oblikovati_persistence::file_access::DocumentFiles,
>(
    files: &mut oblikovati_session::files::FileWorkspace<O, W, F>,
    owner: &mut dyn crate::PartTopologyAccess<O, W, F>,
    document: u64,
    label: &str,
    edit: impl FnOnce(
        &mut DocumentSnapshot,
        &mut oblikovati_model::history::RecipeReservations<'_>,
    ) -> Result<R, Error>,
    policy: PrismResolutionPolicy,
    replay: CheckpointReplay,
) -> Result<(R, bool), Error> {
    let ((reply, candidate), transition) = files
        .commands(document)
        .map_err(|error| Error(error.to_string()))?
        .run_projected_with_transition(label, edit, |snapshot, reply, history| {
            Ok((reply, replay.prepare(snapshot, history)?))
        })
        .map_err(|error| Error(error.to_string()))?;
    let Some(candidate) = candidate else {
        return Ok((reply, false));
    };
    publish_checkpoint(
        files,
        owner,
        document,
        label,
        policy,
        candidate,
        &transition,
    )?;
    Ok((reply, true))
}

fn publish_checkpoint<
    O: oblikovati_session::TransactionObserver,
    W: oblikovati_session::workspace::WorkspaceObserver,
    F: oblikovati_persistence::file_access::DocumentFiles,
>(
    files: &mut oblikovati_session::files::FileWorkspace<O, W, F>,
    owner: &mut dyn crate::PartTopologyAccess<O, W, F>,
    document: u64,
    label: &str,
    policy: PrismResolutionPolicy,
    candidate: NamedCandidate,
    transition: &EvaluationTransition,
) -> Result<(), Error> {
    owner
        .adopt_checkpoints(
            files,
            document,
            crate::CertifiedCheckpointCandidate { policy, candidate },
            transition,
        )
        .map_err(|error| {
            Error(format!(
                "document {document} edit {label:?} committed but replay adoption failed: {error}"
            ))
        })
}

fn replay_for_prefix<
    O: oblikovati_session::TransactionObserver,
    W: oblikovati_session::workspace::WorkspaceObserver,
>(
    workspace: &oblikovati_session::workspace::WorkspaceSession<O, W>,
    document: u64,
    policy: Option<PrismResolutionPolicy>,
    prefix: NamedPrefix,
) -> Option<CheckpointReplay> {
    let guid = workspace.identity(document).ok()?.internal_name;
    Some(CheckpointReplay {
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
