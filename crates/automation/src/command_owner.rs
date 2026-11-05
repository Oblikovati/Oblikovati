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
    fn refresh_after_command(&mut self, id: u64) -> Result<(), Error>;
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
    fn refresh_after_command(&mut self, _id: u64) -> Result<(), Error> {
        Ok(())
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
    fn refresh_after_command(&mut self, id: u64) -> Result<(), Error> {
        if self
            .workspace()
            .session(id)
            .map_err(|error| Error(error.to_string()))?
            .in_transaction()
        {
            return Ok(());
        }
        let report = self
            .refresh_derived_from(id)
            .map_err(|error| refresh_error(id, error))?;
        check_refresh_report(report).map_err(|reason| refresh_error(id, reason))
    }
}

pub(super) fn edit(
    owner: &mut impl CommandOwner,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let edit = PreparedEdit::decode(method, request)?;
    let id = edit.document(owner.workspace().active_document())?;
    let response = owner
        .commands(id)
        .map_err(|error| Error(error.to_string()))?
        .run(edit.label(), |snapshot, reservations| {
            // Encode before committing, so a response failure cannot leave an unreported edit.
            edit.apply(snapshot, reservations)
        })
        .map_err(|error| Error(error.to_string()))?;
    owner.refresh_after_command(id)?;
    Ok(response)
}

fn refresh_error(id: u64, reason: impl std::fmt::Display) -> Error {
    Error(format!(
        "command completed for document {id}, but derived refresh is incomplete: {reason}"
    ))
}

pub(super) fn check_refresh_report(
    report: oblikovati_session::files::DerivedRefreshReport,
) -> Result<(), String> {
    if report.failures.is_empty() && report.blocked.is_empty() {
        return Ok(());
    }
    Err(format!(
        "failed tables {:?}; blocked readers {:?}",
        report.failures, report.blocked
    ))
}
