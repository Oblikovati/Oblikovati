// SPDX-License-Identifier: GPL-2.0-only
//! Preserve clean part checkpoints across an accepted document edit.
use crate::{command_owner::CommandOwner, edit::PreparedEdit};
use oblikovati_api::Error;
use oblikovati_martensite::PrismResolutionPolicy;
use oblikovati_model::history::{DocumentSnapshot, EvaluationTransition};

type NamedCandidate = oblikovati_session::OwnedCheckpointCandidate;
type CheckpointReplay = oblikovati_session::OwnedCheckpointReplay;

fn current_prefix(owner: &impl CommandOwner, document: u64) -> Option<CheckpointReplay> {
    let workspace = owner.workspace();
    let session = workspace.session(document).ok()?;
    if session.in_transaction() {
        return None;
    }
    let (policy, prefix) = session.owned_checkpoint_prefix()?;
    workspace
        .owned_checkpoint_replay(document, policy, prefix)
        .ok()
}
pub(super) fn certified_prefix(
    owner: &impl CommandOwner,
    document: u64,
    prefix: crate::CertifiedCheckpointPrefix,
) -> Result<Option<CheckpointReplay>, Error> {
    let workspace = owner.workspace();
    let session = workspace
        .session(document)
        .map_err(|error| Error(error.to_string()))?;
    if session.in_transaction() {
        return Ok(None);
    }
    workspace
        .owned_checkpoint_replay(document, prefix.policy, prefix.prefix)
        .map(Some)
        .map_err(Error)
}

/// Commit an edit and publish a matching checkpoint when its prefix is safe.
/// Example: `oblikovati_automation::edit_part_with_checkpoints(files, owner, document, label, edit)?`.
pub fn edit_with_owner<
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
    let replay = match prefix.filter(|prefix| prefix.policy == policy) {
        Some(prefix) => certified_prefix(files, document, prefix)?,
        None => None,
    };
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
            Ok((reply, replay.prepare(snapshot, history).map_err(Error)?))
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

fn adopt_parameter_candidate(
    owner: &mut impl CommandOwner,
    replay: &CheckpointReplay,
    candidate: NamedCandidate,
    transition: &EvaluationTransition,
) -> Result<(), Error> {
    let document = replay.document();
    let session = owner
        .workspace()
        .session(document)
        .map_err(|error| Error(error.to_string()))?;
    let completed = replay
        .complete(session, candidate, transition)
        .map_err(Error)?;
    owner
        .commands(document)
        .map_err(|error| Error(error.to_string()))?
        .adopt_owned_replayed_checkpoints(replay.policy(), completed)
        .map_err(Error)
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
                Ok::<_, Error>((response, replay.prepare(snapshot, history).map_err(Error)?))
            },
        )
        .map_err(|error| Error(error.to_string()))?;
    if let Some(candidate) = candidate {
        adopt_parameter_candidate(owner, &replay, candidate, &transition).map_err(|error| {
            Error(format!(
                "document {document}: parameter edit committed but replay adoption failed: {error}"
            ))
        })?;
    }
    owner.refresh_after_command(document)?;
    Ok(response)
}
