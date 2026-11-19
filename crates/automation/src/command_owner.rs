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
    fn verify_named_candidate(
        &self,
        id: u64,
        candidate: &oblikovati_model::history::DocumentSnapshot,
    ) -> Result<(), String>;
    fn navigate_named_prepared(
        &mut self,
        id: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        direction: oblikovati_model::history::PreparedNavigation,
    ) -> Result<(), String>;
    fn run_named_prepared<R, T>(
        &mut self,
        id: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        edit: impl FnOnce(
            &mut oblikovati_model::history::DocumentSnapshot,
            &mut oblikovati_model::history::RecipeReservations<'_>,
        ) -> Result<R, String>,
        project: impl FnOnce(&oblikovati_model::history::DocumentSnapshot, R, u64) -> Result<T, String>,
    ) -> Result<T, String>;
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
    fn verify_named_candidate(
        &self,
        id: u64,
        candidate: &oblikovati_model::history::DocumentSnapshot,
    ) -> Result<(), String> {
        self.verify_unlinked_named_candidate(id, candidate)
    }
    fn navigate_named_prepared(
        &mut self,
        id: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        direction: oblikovati_model::history::PreparedNavigation,
    ) -> Result<(), String> {
        self.navigate_unlinked_prepared(id, policy, direction)
    }
    fn run_named_prepared<R, T>(
        &mut self,
        id: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        edit: impl FnOnce(
            &mut oblikovati_model::history::DocumentSnapshot,
            &mut oblikovati_model::history::RecipeReservations<'_>,
        ) -> Result<R, String>,
        project: impl FnOnce(&oblikovati_model::history::DocumentSnapshot, R, u64) -> Result<T, String>,
    ) -> Result<T, String> {
        self.run_unlinked_prepared(id, policy, edit, project)
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
    fn verify_named_candidate(
        &self,
        id: u64,
        candidate: &oblikovati_model::history::DocumentSnapshot,
    ) -> Result<(), String> {
        FileWorkspace::verify_named_candidate(self, id, candidate)
    }
    fn navigate_named_prepared(
        &mut self,
        id: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        direction: oblikovati_model::history::PreparedNavigation,
    ) -> Result<(), String> {
        FileWorkspace::navigate_named_prepared(self, id, policy, direction)
    }
    fn run_named_prepared<R, T>(
        &mut self,
        id: u64,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
        edit: impl FnOnce(
            &mut oblikovati_model::history::DocumentSnapshot,
            &mut oblikovati_model::history::RecipeReservations<'_>,
        ) -> Result<R, String>,
        project: impl FnOnce(&oblikovati_model::history::DocumentSnapshot, R, u64) -> Result<T, String>,
    ) -> Result<T, String> {
        FileWorkspace::run_named_prepared(self, id, policy, edit, project)
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
    if method == oblikovati_api::wire::METHOD_MODEL_STATES_SET_TABLE {
        return crate::model_states::table_apply::set(owner, crate::decode(method, request)?);
    }
    let edit = PreparedEdit::decode(method, request)?;
    let id = edit.document(owner.workspace().active_document())?;
    if edit.requires_checkpoint_replay() {
        return crate::checkpoint_replay::try_edit(owner, edit, id);
    }
    apply_edit(owner, edit, id)
}

pub(super) fn apply_edit(
    owner: &mut impl CommandOwner,
    edit: PreparedEdit,
    id: u64,
) -> Result<Vec<u8>, Error> {
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
