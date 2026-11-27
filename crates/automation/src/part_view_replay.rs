// SPDX-License-Identifier: GPL-2.0-only
//! Preserve certified replay results across view edits and history navigation.
use oblikovati_api::Error;
use oblikovati_model::history::{DocumentSnapshot, EvaluationTransition, RecipeReservations};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    PartViewNavigationDirection, TransactionObserver, files::FileWorkspace,
    workspace::WorkspaceObserver,
};
type ReplayPolicy = Option<oblikovati_martensite::PrismResolutionPolicy>;

pub(crate) fn edit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    policy: ReplayPolicy,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let edit = crate::edit::PreparedEdit::decode(method, request)?;
    let document = edit.document(files.workspace().active_document())?;
    run(
        files,
        document,
        Some(policy),
        edit.label(),
        |snapshot, ids| edit.apply(snapshot, ids),
    )
}

pub(crate) fn run<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    policy: Option<ReplayPolicy>,
    label: &str,
    edit: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<Vec<u8>, Error>,
) -> Result<Vec<u8>, Error> {
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let replay = policy
        .filter(|_| !session.in_transaction())
        .and_then(|policy| {
            files
                .workspace()
                .owned_part_replay(document, policy)
                .ok()
                .map(|replay| (policy, replay))
        });
    let Some((policy, replay)) = replay else {
        return files
            .commands(document)
            .map_err(|cause| Error(cause.to_string()))?
            .run(label, edit)
            .map_err(|cause| Error(cause.to_string()));
    };
    let ((response, candidate), transition) = files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?
        .run_projected_with_transition(label, edit, |candidate, response, history| {
            Ok((response, replay.capture_unchanged(history, candidate)?))
        })
        .map_err(|cause| Error(cause.to_string()))?;
    adopt(files, document, policy, candidate, &transition);
    Ok(response)
}

pub(crate) fn adopt<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    policy: ReplayPolicy,
    candidate: oblikovati_session::OwnedPartCandidate,
    transition: &oblikovati_model::history::EvaluationTransition,
) {
    if let Err(cause) = files.adopt_owned_part_candidate(document, policy, candidate, transition) {
        eprintln!(
            "{}",
            serde_json::json!({"level":"warn", "event":"part_view_replay_adoption_failed", "document":document, "error":cause.to_string()})
        );
    }
}

pub(crate) fn navigate_with_replay<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    document: u64,
    direction: PartViewNavigationDirection,
    pending_sync: &mut bool,
) -> Result<bool, Error> {
    let Some(owner) = topology.as_deref_mut() else {
        return Ok(false);
    };
    let policy = owner.resolution_policy();
    let Ok(replay) = files.workspace().owned_part_replay(document, policy) else {
        return Ok(false);
    };
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let history = session.history();
    let target = match direction {
        PartViewNavigationDirection::Undo => history.undo_candidate(),
        PartViewNavigationDirection::Redo => history.redo_candidate(),
    }
    .map_err(|cause| Error(cause.to_string()))?;
    if session.snapshot().part_replay_change(target).is_none() {
        let candidate = replay.capture_unchanged(history, target)?;
        let transition = accept_navigation(files, document, direction, pending_sync)?;
        adopt(files, document, policy, candidate, &transition);
        return Ok(true);
    }
    let Some((policy, prefix)) = crate::checkpoint_replay::owner_prefix(files, owner, document)?
    else {
        return Ok(false);
    };
    let Some(candidate) = prefix.prepare(target, history).map_err(Error)? else {
        return Ok(false);
    };
    let transition = accept_navigation(files, document, direction, pending_sync)?;
    let label = match direction {
        PartViewNavigationDirection::Undo => "Undo",
        PartViewNavigationDirection::Redo => "Redo",
    };
    crate::checkpoint_replay::publish_checkpoint(
        files,
        owner,
        document,
        label,
        policy,
        candidate,
        &transition,
    )?;
    Ok(true)
}

fn accept_navigation<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    direction: PartViewNavigationDirection,
    pending_sync: &mut bool,
) -> Result<EvaluationTransition, Error> {
    let mut commands = files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let transition = match direction {
        PartViewNavigationDirection::Undo => commands.undo_with_transition(),
        PartViewNavigationDirection::Redo => commands.redo_with_transition(),
    }
    .map_err(|cause| Error(cause.to_string()))?;
    *pending_sync = true;
    Ok(transition)
}
