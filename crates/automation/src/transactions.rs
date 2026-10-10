// SPDX-License-Identifier: GPL-2.0-only

use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

/// Transaction commands use the session history directly, without opening an automatic edit.
pub const TRANSACTION_METHODS: [&str; 6] = [
    METHOD_TRANSACTION_BEGIN,
    METHOD_TRANSACTION_JUMP_TO,
    METHOD_TRANSACTION_END,
    METHOD_TRANSACTION_ABORT,
    METHOD_TRANSACTION_UNDO,
    METHOD_TRANSACTION_REDO,
];

pub(super) fn state<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<UndoState, Error> {
    let Some(id) = workspace.active_document() else {
        return Ok(UndoState::default());
    };
    let session = workspace
        .session(id)
        .map_err(|error| Error(error.to_string()))?;
    let undo = session.undo_labels();
    let redo = session.redo_labels();
    Ok(UndoState {
        can_undo: !undo.is_empty(),
        can_redo: !redo.is_empty(),
        next_undo: undo.last().copied().unwrap_or_default().into(),
        next_redo: redo.first().copied().unwrap_or_default().into(),
    })
}

pub(super) fn apply(
    workspace: &mut impl crate::command_owner::CommandOwner,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    if method == METHOD_TRANSACTION_JUMP_TO {
        return jump(workspace, decode(method, request)?);
    }
    let id = target_document(workspace.workspace(), 0)?;
    let mut commands = workspace
        .commands(id)
        .map_err(|error| Error(error.to_string()))?;
    if method == METHOD_TRANSACTION_BEGIN {
        let request: TransactionBeginArgs = decode(method, request)?;
        commands
            .begin(&request.label)
            .map_err(|error| Error(error.to_string()))?;
        return encode(&OKResult { ok: true });
    }
    match method {
        METHOD_TRANSACTION_END => commands.end(),
        METHOD_TRANSACTION_ABORT => commands.abort(),
        METHOD_TRANSACTION_UNDO => commands.undo(),
        METHOD_TRANSACTION_REDO => commands.redo(),
        _ => {
            return Err(Error(format!(
                "transaction method {method:?}: expected one of {TRANSACTION_METHODS:?}"
            )));
        }
    }
    .map_err(|error| Error(error.to_string()))?;
    encode(&state(workspace.workspace())?)
}

pub(super) fn history<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    request: TransactionHistoryArgs,
) -> Result<TransactionHistory, Error> {
    let id = target_document(workspace, request.document)?;
    let session = workspace
        .session(id)
        .map_err(|error| Error(error.to_string()))?;
    let position = i64::try_from(session.undo_labels().len()).map_err(|error| {
        Error(format!(
            "history position: expected a signed 64-bit count: {error}"
        ))
    })?;
    Ok(TransactionHistory {
        document: id,
        name: workspace
            .display_name(id)
            .map_err(|error| Error(error.to_string()))?
            .into(),
        position,
        entries: Some(session.history_entries()),
    })
}

fn target_document<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
    requested: u64,
) -> Result<u64, Error> {
    if requested != 0 {
        return Ok(requested);
    }
    workspace
        .active_document()
        .ok_or_else(|| Error("history document 0: expected an active document".into()))
}

fn jump(
    workspace: &mut impl crate::command_owner::CommandOwner,
    request: TransactionJumpToArgs,
) -> Result<Vec<u8>, Error> {
    let id = target_document(workspace.workspace(), request.document)?;
    workspace
        .commands(id)
        .map_err(|error| Error(error.to_string()))?
        .jump_to(request.position)
        .map_err(|error| Error(error.to_string()))?;
    encode(&history(
        workspace.workspace(),
        TransactionHistoryArgs { document: id },
    )?)
}
