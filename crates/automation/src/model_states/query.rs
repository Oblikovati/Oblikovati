// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::{
    TransactionObserver,
    workspace::{WorkspaceObserver, WorkspaceSession},
};
pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver>(
    workspace: &WorkspaceSession<O, W>,
) -> Result<ModelStatesResult, Error> {
    let id = workspace
        .active_document()
        .ok_or_else(|| Error("model-state list: expected an active part document".into()))?;
    let session = workspace
        .session(id)
        .map_err(|cause| Error(cause.to_string()))?;
    let members = session
        .snapshot()
        .model_states
        .as_ref()
        .ok_or_else(|| Error(format!("document {id}: expected part model-state content")))?;
    Ok(member_list(members))
}
