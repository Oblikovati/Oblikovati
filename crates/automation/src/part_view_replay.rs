// SPDX-License-Identifier: GPL-2.0-only
//! Preserve certified named bodies across a proven view-only edit.
use oblikovati_api::Error;
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
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

pub(crate) fn navigate_unchanged<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    document: u64,
    direction: oblikovati_session::PartViewNavigationDirection,
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
        oblikovati_session::PartViewNavigationDirection::Undo => history.undo_candidate(),
        oblikovati_session::PartViewNavigationDirection::Redo => history.redo_candidate(),
    }
    .map_err(|cause| Error(cause.to_string()))?;
    if session.snapshot().part_replay_change(target).is_some() {
        return Ok(false);
    }
    let candidate = replay.capture_unchanged(history, target)?;
    let mut commands = files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let transition = match direction {
        oblikovati_session::PartViewNavigationDirection::Undo => commands.undo_with_transition(),
        oblikovati_session::PartViewNavigationDirection::Redo => commands.redo_with_transition(),
    }
    .map_err(|cause| Error(cause.to_string()))?;
    adopt(files, document, policy, candidate, &transition);
    Ok(true)
}
