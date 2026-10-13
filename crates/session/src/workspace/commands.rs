// SPDX-License-Identifier: GPL-2.0-only

use crate::{CommandError, DocumentSession, SessionError, TransactionObserver};
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};

/// A borrowed command capability. It cannot replace a workspace's document owner.
pub struct SessionCommands<'a, O> {
    pub(super) session: &'a mut DocumentSession<O>,
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Start a bounded group on this document; example: `commands.begin("batch")?;`.
    pub fn begin(&mut self, label: &str) -> Result<(), SessionError> {
        self.session.begin(label)
    }

    /// Finish one bounded level on this document; example: `commands.end()?;`.
    pub fn end(&mut self) -> Result<(), SessionError> {
        self.session.end()
    }

    /// Cancel this document's whole group; example: `commands.abort()?;`.
    pub fn abort(&mut self) -> Result<(), SessionError> {
        self.session.abort()
    }

    /// Undo this document's latest step; example: `commands.undo()?;`.
    pub fn undo(&mut self) -> Result<(), SessionError> {
        self.session.undo()
    }

    /// Redo this document's next step; example: `commands.redo()?;`.
    pub fn redo(&mut self) -> Result<(), SessionError> {
        self.session.redo()
    }

    /// Run through this document's normal transaction path.
    /// Example: `commands.run("width", |snapshot, _| snapshot.parameters.set_expression(1, "2 cm"))?;`.
    pub fn run<R, E>(
        &mut self,
        label: &str,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.session.run(label, command)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Move this document's cursor without activating it; each step can be vetoed.
    /// Example: `commands.jump_to(0)?` navigates to the opened recipe.
    pub fn jump_to(&mut self, position: i64) -> Result<(), SessionError> {
        self.session.jump_to(position)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Publish a source consequence without an authored transaction.
    /// Example: `commands.refresh_derived_table(7, source.as_ref())?`.
    pub fn refresh_derived_table(
        &mut self,
        table: i64,
        source: Option<&std::collections::BTreeMap<String, oblikovati_model::quantity::Quantity>>,
    ) -> Result<bool, CommandError<oblikovati_model::parameter::ParameterError<u64>>> {
        self.session.refresh_derived_table(table, source)
    }
}

impl<O: TransactionObserver> SessionCommands<'_, O> {
    /// Change view state for a resolved body without editing history or saved content.
    /// Example: `commands.set_body_visible(current_body_key, false)?;` returns whether display changed.
    pub fn set_body_visible(&mut self, key: &str, visible: bool) -> Result<bool, String> {
        self.session.body_visibility.set(key, visible)
    }
}
