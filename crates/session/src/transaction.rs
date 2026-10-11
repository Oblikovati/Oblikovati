// SPDX-License-Identifier: GPL-2.0-only

use crate::sketch_recompute::{SketchUpdateMode, apply_recipe_edit};
use crate::{BoundedGroup, CommandError, DocumentSession, SessionError, TransactionObserver};
use oblikovati_api::{
    types::TRANSACTION_POINT_CURRENT,
    wire::{EVENT_TRANSACTION_ABORTED, EVENT_TRANSACTION_COMMITTED},
};
use oblikovati_model::history::{DocumentSnapshot, RecipeReservations};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Start or nest one bounded undo group; example: `session.begin("Edit Parameters")?;`.
    pub fn begin(&mut self, label: &str) -> Result<(), SessionError> {
        if let Some(group) = &mut self.group {
            group.depth = group
                .depth
                .checked_add(1)
                .ok_or(SessionError::DepthExhausted)?;
            return Ok(());
        }
        self.group = Some(BoundedGroup {
            token: self.history.begin(label)?,
            label: label.into(),
            depth: 1,
        });
        Ok(())
    }

    /// Edit an open group and refresh exposed properties atomically.
    /// Example: `session.edit(|snapshot, ids| add_parameter(snapshot, ids))?;`.
    pub fn edit<R, E>(
        &mut self,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
    ) -> Result<R, CommandError<E>> {
        self.edit_with_recompute(command, SketchUpdateMode::Automatic)
    }
    pub(crate) fn edit_with_recompute<R, E>(
        &mut self,
        command: impl FnOnce(&mut DocumentSnapshot, &mut RecipeReservations<'_>) -> Result<R, E>,
        mode: SketchUpdateMode,
    ) -> Result<R, CommandError<E>> {
        let group = self
            .group
            .as_ref()
            .ok_or(CommandError::Session(SessionError::NoGroup))?;
        let options = mode.options(self.sketch_options);
        self.history
            .edit(&group.token, |snapshot, ids| {
                apply_recipe_edit(snapshot, ids, command, options)
            })
            .map_err(CommandError::Edit)
    }

    /// Close one level; only the outer end records a step and emits events.
    /// Example: `session.end()?;`.
    pub fn end(&mut self) -> Result<(), SessionError> {
        let group = self.group.as_mut().ok_or(SessionError::NoGroup)?;
        if group.depth > 1 {
            group.depth -= 1;
            return Ok(());
        }
        self.end_outer_group()
    }

    fn end_outer_group(&mut self) -> Result<(), SessionError> {
        let group = self.group.as_ref().expect("validated group");
        if self.history.discard_unchanged(&group.token)? {
            self.group = None;
            return Ok(());
        }
        self.history.validate_abort(&group.token)?;
        let label = group.label.clone();
        let event = self.payload(
            EVENT_TRANSACTION_COMMITTED,
            &label,
            TRANSACTION_POINT_CURRENT,
        );
        self.commit_group(event)
    }

    fn commit_group(
        &mut self,
        event: oblikovati_api::wire::TransactionEventPayload,
    ) -> Result<(), SessionError> {
        if let Err(veto) = self.notify_before(&event) {
            self.abort()?;
            return Err(veto);
        }
        let group = self.group.as_ref().expect("validated group");
        self.history.commit(&group.token, false)?;
        self.group = None;
        self.notify_changed(&event);
        Ok(())
    }

    /// Cancel the whole bounded group, including nested levels; example: `session.abort()?;`.
    pub fn abort(&mut self) -> Result<(), SessionError> {
        let group = self.group.as_ref().ok_or(SessionError::NoGroup)?;
        self.history.abort(&group.token)?;
        let label = group.label.clone();
        self.group = None;
        let event = self.payload(EVENT_TRANSACTION_ABORTED, &label, TRANSACTION_POINT_CURRENT);
        self.notify_after(&event);
        Ok(())
    }
}
