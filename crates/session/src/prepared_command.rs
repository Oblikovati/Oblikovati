// SPDX-License-Identifier: GPL-2.0-only
//! Validate a private edit and its observers before publishing any state (#4122).

use crate::{CommandError, DocumentSession, SessionError, TransactionObserver};
use oblikovati_api::{types::TRANSACTION_POINT_CURRENT, wire::EVENT_TRANSACTION_COMMITTED};
use oblikovati_model::history::{
    DocumentSnapshot, EvaluationTransition, PreparedPrivateEdit, PreparedPublication,
    RecipeReservations,
};

impl<O: TransactionObserver> DocumentSession<O> {
    pub(crate) fn run_prepared<R, P: PreparedPublication>(
        &mut self,
        label: &str,
        kind: oblikovati_model::history::HistoryTransitionKind,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, String>,
        prepare: impl FnOnce(&PreparedPrivateEdit<'_>, R) -> Result<P, String>,
        publish: impl FnOnce(&mut Self, P::Output, EvaluationTransition),
    ) -> Result<(), CommandError<String>> {
        if self.in_transaction() {
            return Err(CommandError::Session(SessionError::NavigationGroupOpen));
        }
        let event = self.payload(
            EVENT_TRANSACTION_COMMITTED,
            label,
            TRANSACTION_POINT_CURRENT,
        );
        let options = Some(self.sketch_options);
        let (mut prepared, result) = self
            .history
            .prepare_private_edit(label, |snapshot, ids, _| {
                crate::sketch_recompute::apply_recipe_edit(
                    snapshot,
                    ids,
                    command,
                    options,
                    |_, value| Ok(value),
                )
            })
            .map_err(CommandError::Edit)?;
        prepared.set_transition_kind(kind);
        let publication = prepare(&prepared, result).map_err(prepared_error)?;
        publication.validate(&prepared).map_err(prepared_error)?;
        publication
            .validate_target(&prepared.source().evaluation_scope())
            .map_err(prepared_error)?;
        let changed = prepared.changed();
        if changed {
            self.observer
                .before(&event, prepared.snapshot())
                .map_err(|error| CommandError::Session(SessionError::Veto(error)))?;
        }
        let (result, transition) =
            oblikovati_model::history::publish_prepared(prepared, publication)
                .map_err(prepared_error)?;
        publish(self, result, transition);
        if changed {
            self.notify_changed(&event);
        }
        Ok(())
    }
}

fn prepared_error(error: String) -> CommandError<String> {
    CommandError::Edit(oblikovati_model::history::EditError::Mutation(
        crate::MutationError::Command(error),
    ))
}

#[cfg(test)]
#[allow(dead_code)]
#[path = "../../../test-utilities/rust/selected_sketch_prism.rs"]
pub(crate) mod fixture;
#[cfg(test)]
#[path = "prepared_command_tests.rs"]
mod tests;

impl<O: TransactionObserver> DocumentSession<O> {
    pub(crate) fn navigate_prepared<P: PreparedPublication>(
        &mut self,
        direction: oblikovati_model::history::PreparedNavigation,
        prepare: impl FnOnce(&PreparedPrivateEdit<'_>) -> Result<P, String>,
        publish: impl FnOnce(&mut Self, P::Output, EvaluationTransition),
    ) -> Result<(), String> {
        use oblikovati_api::{
            types::{TRANSACTION_POINT_NEXT, TRANSACTION_POINT_PREVIOUS},
            wire::{EVENT_TRANSACTION_REDONE, EVENT_TRANSACTION_UNDONE},
        };
        use oblikovati_model::history::PreparedNavigation;
        let (label, kind, point, sketch) = match direction {
            PreparedNavigation::Undo => (
                self.history
                    .undo_label()
                    .map_err(|error| error.to_string())?
                    .to_owned(),
                EVENT_TRANSACTION_UNDONE,
                TRANSACTION_POINT_PREVIOUS,
                self.history
                    .undo_sketch_transition()
                    .map(|value| crate::SketchEditNavigation {
                        from: value.after,
                        to: value.before,
                    }),
            ),
            PreparedNavigation::Redo => (
                self.history
                    .redo_label()
                    .map_err(|error| error.to_string())?
                    .to_owned(),
                EVENT_TRANSACTION_REDONE,
                TRANSACTION_POINT_NEXT,
                self.history
                    .redo_sketch_transition()
                    .map(|value| crate::SketchEditNavigation {
                        from: value.before,
                        to: value.after,
                    }),
            ),
        };
        let event = self.payload(kind, &label, point);
        let prepared = self
            .history
            .prepare_private_navigation(direction)
            .map_err(|error| error.to_string())?;
        let publication = prepare(&prepared)?;
        publication.validate(&prepared)?;
        publication.validate_target(&prepared.source().evaluation_scope())?;
        self.observer.before(&event, prepared.source().snapshot())?;
        let (result, transition) =
            oblikovati_model::history::publish_prepared(prepared, publication)?;
        publish(self, result, transition);
        self.remember_sketch_navigation(sketch);
        self.notify_changed(&event);
        Ok(())
    }
}
