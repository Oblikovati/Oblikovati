// SPDX-License-Identifier: GPL-2.0-only

use crate::edit::PreparedEdit;
use oblikovati_api::Error;
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    TransactionObserver,
    files::FileWorkspace,
    workspace::{SessionCommands, WorkspaceError, WorkspaceObserver, WorkspaceSession},
};

pub(super) trait CommandOwner {
    type Observer: TransactionObserver;
    type WorkspaceObserver: WorkspaceObserver;
    fn workspace(&self) -> &WorkspaceSession<Self::Observer, Self::WorkspaceObserver>;
    fn commands(&mut self, id: u64) -> Result<SessionCommands<'_, Self::Observer>, WorkspaceError>;
}
impl<O: TransactionObserver, W: WorkspaceObserver> CommandOwner for WorkspaceSession<O, W> {
    type Observer = O;
    type WorkspaceObserver = W;
    fn workspace(&self) -> &WorkspaceSession<O, W> {
        self
    }
    fn commands(&mut self, id: u64) -> Result<SessionCommands<'_, O>, WorkspaceError> {
        self.commands(id)
    }
}
impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> CommandOwner
    for FileWorkspace<O, W, F>
{
    type Observer = O;
    type WorkspaceObserver = W;
    fn workspace(&self) -> &WorkspaceSession<O, W> {
        self.workspace()
    }
    fn commands(&mut self, id: u64) -> Result<SessionCommands<'_, O>, WorkspaceError> {
        self.commands(id)
    }
}

pub(super) fn edit(
    owner: &mut impl CommandOwner,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let edit = PreparedEdit::decode(method, request)?;
    let id = edit.document(owner.workspace().active_document())?;
    owner
        .commands(id)
        .map_err(|error| Error(error.to_string()))?
        .run(edit.label(), |snapshot, reservations| {
            // Encode before committing, so a response failure cannot leave an unreported edit.
            edit.apply(snapshot, reservations)
        })
        .map_err(|error| Error(error.to_string()))
}
