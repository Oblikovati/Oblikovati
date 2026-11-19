// SPDX-License-Identifier: GPL-2.0-only
//! Route tagged view history through the current topology and display (#4118).
use oblikovati_api::{Error, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    PartViewNavigationDirection, TransactionObserver, files::FileWorkspace,
    workspace::WorkspaceObserver,
};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    display: &mut Option<&mut dyn crate::PartViewDisplayAccess>,
    method: &str,
    request: &[u8],
) -> Result<Option<(Vec<u8>, bool)>, Error> {
    if method == METHOD_TRANSACTION_JUMP_TO {
        let request: TransactionJumpToArgs = crate::decode(method, request)?;
        if !crosses_view(files, &request)? {
            return Ok(None);
        }
        return jump(files, topology, display, request).map(Some);
    }
    let direction = match method {
        METHOD_TRANSACTION_UNDO => PartViewNavigationDirection::Undo,
        METHOD_TRANSACTION_REDO => PartViewNavigationDirection::Redo,
        _ => return Ok(None),
    };
    let Some(document) = files.workspace().active_document() else {
        return Ok(None);
    };
    if !is_tagged(files, document, direction)? {
        return Ok(None);
    }
    crate::part_view_display::navigate(files, topology, display, document, direction)?;
    Ok(Some((
        crate::encode(&crate::transactions::state(files.workspace())?)?,
        false,
    )))
}
fn crosses_view<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: &TransactionJumpToArgs,
) -> Result<bool, Error> {
    let document = if request.document == 0 {
        files
            .workspace()
            .active_document()
            .ok_or_else(|| Error("history document 0: expected an active document".into()))?
    } else {
        request.document
    };
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let Ok(destination) = usize::try_from(request.position) else {
        return Ok(false);
    };
    if destination > session.undo_labels().len() + session.redo_labels().len() {
        return Ok(false);
    }
    Ok(session.history().jump_crosses_part_view(destination))
}

fn is_tagged<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    direction: PartViewNavigationDirection,
) -> Result<bool, Error> {
    let history = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?
        .history();
    Ok(match direction {
        PartViewNavigationDirection::Undo => history.undo_part_view_display().is_some(),
        PartViewNavigationDirection::Redo => history.redo_part_view_display().is_some(),
    })
}

fn jump<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    display: &mut Option<&mut dyn crate::PartViewDisplayAccess>,
    request: TransactionJumpToArgs,
) -> Result<(Vec<u8>, bool), Error> {
    let document = if request.document == 0 {
        files
            .workspace()
            .active_document()
            .ok_or_else(|| Error("history document 0: expected an active document".into()))?
    } else {
        request.document
    };
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| Error(cause.to_string()))?;
    let mut cursor = session.undo_labels().len();
    let total = cursor + session.redo_labels().len();
    let destination = usize::try_from(request.position)
        .ok()
        .filter(|position| *position <= total)
        .ok_or_else(|| {
            Error(format!(
                "history position {}: expected a cursor in 0..={total}",
                request.position
            ))
        })?;
    if session.in_transaction() {
        return Err(Error(
            "history jump with an open group: expected all transactions closed".into(),
        ));
    }
    let mut needs_reference_sync = false;
    while cursor != destination {
        let direction = if cursor > destination {
            PartViewNavigationDirection::Undo
        } else {
            PartViewNavigationDirection::Redo
        };
        needs_reference_sync |= step(files, topology, display, document, direction)?;
        if direction == PartViewNavigationDirection::Undo {
            cursor -= 1;
        } else {
            cursor += 1;
        }
    }
    Ok((
        crate::encode(&crate::transactions::history(
            files.workspace(),
            TransactionHistoryArgs { document },
        )?)?,
        needs_reference_sync,
    ))
}
fn step<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    topology: &mut Option<&mut dyn crate::PartTopologyAccess<O, W, F>>,
    display: &mut Option<&mut dyn crate::PartViewDisplayAccess>,
    document: u64,
    direction: PartViewNavigationDirection,
) -> Result<bool, Error> {
    if is_tagged(files, document, direction)? {
        crate::part_view_display::navigate(files, topology, display, document, direction)?;
        return Ok(false);
    }
    let mut commands = files
        .commands(document)
        .map_err(|cause| Error(cause.to_string()))?;
    match direction {
        PartViewNavigationDirection::Undo => commands.undo(),
        PartViewNavigationDirection::Redo => commands.redo(),
    }
    .map_err(|cause| Error(cause.to_string()))?;
    crate::command_owner::CommandOwner::refresh_after_command(files, document)?;
    Ok(true)
}
