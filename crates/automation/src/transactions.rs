// SPDX-License-Identifier: GPL-2.0-only

use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};

/// Transaction commands use the session history directly, without opening an automatic edit.
pub const TRANSACTION_METHODS: [&str; 5] = [
    METHOD_TRANSACTION_BEGIN,
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

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &mut WorkspaceSession<O, W>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let mut commands = workspace
        .active_commands()
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
    encode(&state(workspace)?)
}
