// SPDX-License-Identifier: GPL-2.0-only

use crate::{DocumentSession, SessionError, TransactionObserver};
use oblikovati_api::{
    types::{TRANSACTION_POINT_NEXT, TRANSACTION_POINT_PREVIOUS, TRANSACTION_POINT_UNKNOWN},
    wire::{EVENT_TRANSACTION_DELETED, EVENT_TRANSACTION_REDONE, EVENT_TRANSACTION_UNDONE},
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Undo after observers accept the move; example: `session.undo()?;`.
    pub fn undo(&mut self) -> Result<(), SessionError> {
        self.history.undo_label()?;
        if self.history.undo_part_view_display().is_some() {
            return Err(SessionError::PartViewDisplayOwner);
        }
        self.undo_display_checked()
    }
    /// Undo an ordinary step and return its accepted generation transition.
    /// Example: `session.undo_with_transition()?` after capturing a candidate.
    pub fn undo_with_transition(
        &mut self,
    ) -> Result<oblikovati_model::history::EvaluationTransition, SessionError> {
        self.history.undo_label()?;
        if self.history.undo_part_view_display().is_some() {
            return Err(SessionError::PartViewDisplayOwner);
        }
        self.undo_display_transition()
    }

    pub(super) fn undo_display_checked(&mut self) -> Result<(), SessionError> {
        self.undo_display_transition().map(|_| ())
    }

    pub(super) fn undo_display_transition(
        &mut self,
    ) -> Result<oblikovati_model::history::EvaluationTransition, SessionError> {
        let label = self.history.undo_label()?;
        let event = self.payload(EVENT_TRANSACTION_UNDONE, label, TRANSACTION_POINT_PREVIOUS);
        self.notify_before(&event)?;
        let transition = self.history.undo_sketch_transition();
        let accepted = self.history.undo_with_transition()?;
        self.remember_sketch_navigation(transition.map(|value| crate::SketchEditNavigation {
            from: value.after,
            to: value.before,
        }));
        self.notify_changed(&event);
        Ok(accepted)
    }

    /// Redo after observers accept the move; example: `session.redo()?;`.
    pub fn redo(&mut self) -> Result<(), SessionError> {
        self.history.redo_label()?;
        if self.history.redo_part_view_display().is_some() {
            return Err(SessionError::PartViewDisplayOwner);
        }
        self.redo_display_checked()
    }
    /// Redo an ordinary step and return its accepted generation transition.
    /// Example: `session.redo_with_transition()?` after capturing a candidate.
    pub fn redo_with_transition(
        &mut self,
    ) -> Result<oblikovati_model::history::EvaluationTransition, SessionError> {
        self.history.redo_label()?;
        if self.history.redo_part_view_display().is_some() {
            return Err(SessionError::PartViewDisplayOwner);
        }
        self.redo_display_transition()
    }

    pub(super) fn redo_display_checked(&mut self) -> Result<(), SessionError> {
        self.redo_display_transition().map(|_| ())
    }

    pub(super) fn redo_display_transition(
        &mut self,
    ) -> Result<oblikovati_model::history::EvaluationTransition, SessionError> {
        let label = self.history.redo_label()?;
        let event = self.payload(EVENT_TRANSACTION_REDONE, label, TRANSACTION_POINT_NEXT);
        self.notify_before(&event)?;
        let transition = self.history.redo_sketch_transition();
        let accepted = self.history.redo_with_transition()?;
        self.remember_sketch_navigation(transition.map(|value| crate::SketchEditNavigation {
            from: value.before,
            to: value.after,
        }));
        self.notify_changed(&event);
        Ok(accepted)
    }

    /// Drop this document's stream and return its observer; example: `let observer = session.close();`.
    pub fn close(mut self) -> O {
        let event = self.payload(EVENT_TRANSACTION_DELETED, "", TRANSACTION_POINT_UNKNOWN);
        self.notify_after(&event);
        self.observer
    }
}

impl<O: TransactionObserver> DocumentSession<O> {
    /// Move through ordinary undo/redo steps; a veto stops at the last accepted step.
    /// Example: `session.jump_to(0)?` restores the initial recipe without dropping redo.
    pub fn jump_to(&mut self, position: i64) -> Result<(), SessionError> {
        let destination = self.validate_jump(position)?;
        let mut cursor = self.undo_labels().len();
        while cursor > destination {
            self.undo()?;
            cursor -= 1;
        }
        while cursor < destination {
            self.redo()?;
            cursor += 1;
        }
        Ok(())
    }

    fn validate_jump(&self, position: i64) -> Result<usize, SessionError> {
        if self.in_transaction() {
            return Err(SessionError::NavigationGroupOpen);
        }
        let total = self.undo_labels().len() + self.redo_labels().len();
        usize::try_from(position)
            .ok()
            .filter(|cursor| *cursor <= total)
            .ok_or(SessionError::HistoryPosition {
                supplied: position,
                total,
            })
    }
}
