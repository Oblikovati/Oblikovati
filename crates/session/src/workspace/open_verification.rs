// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::EditError;
use oblikovati_solve::nonlinear::SolveOptions;
use std::convert::Infallible;

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    pub(super) fn prepare_entry(
        &self,
        id: u64,
        mut history: DocumentEditHistory,
        observer: O,
    ) -> Result<WorkspaceEntry<O>, WorkspaceError> {
        let baseline = history.snapshot().clone();
        let options = self.sketch_options;
        history
            .refresh::<Infallible>(|snapshot| {
                crate::sketch_recompute::update_pending_sketches(snapshot, options);
                Ok(())
            })
            .map_err(initial_verification_error)?;
        Ok(loaded_entry(id, history, baseline, observer, options))
    }
    pub(super) fn install_entry(
        &mut self,
        id: u64,
        mut entry: WorkspaceEntry<O>,
        identity: IdentityRegistration,
    ) -> RegisteredDocument {
        entry.identity_dirty = identity != IdentityRegistration::Preserved;
        self.entries.insert(id, entry);
        self.order.push(id);
        RegisteredDocument {
            session_id: id,
            identity,
        }
    }
}
fn loaded_entry<O: TransactionObserver>(
    id: u64,
    history: DocumentEditHistory,
    baseline: DocumentSnapshot,
    observer: O,
    options: SolveOptions,
) -> WorkspaceEntry<O> {
    WorkspaceEntry {
        baseline,
        baseline_derived_high_water: history.derived_high_water(),
        baseline_parameter_high_water: history.high_water(),
        baseline_feature_high_water: history.feature_high_water(),
        baseline_sketch_marks: history.sketch_reservations(),
        display_name: String::new(),
        identity_dirty: false,
        session: DocumentSession::new_with_sketch_options(id, history, observer, options),
    }
}
fn initial_verification_error(error: EditError<Infallible>) -> WorkspaceError {
    match error {
        EditError::History(cause) => WorkspaceError::InitialVerification(cause),
        EditError::Mutation(impossible) => match impossible {},
    }
}
