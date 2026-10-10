// SPDX-License-Identifier: GPL-2.0-only

use crate::{DocumentSession, SessionError, TransactionObserver};
use oblikovati_api::{
    types::{TRANSACTION_POINT_NEXT, TRANSACTION_POINT_PREVIOUS, TRANSACTION_POINT_UNKNOWN},
    wire::{EVENT_TRANSACTION_DELETED, EVENT_TRANSACTION_REDONE, EVENT_TRANSACTION_UNDONE},
};

impl<O: TransactionObserver> DocumentSession<O> {
    /// Undo after observers accept the move; example: `session.undo()?;`.
    pub fn undo(&mut self) -> Result<(), SessionError> {
        let label = self.history.undo_label()?;
        let event = self.payload(EVENT_TRANSACTION_UNDONE, label, TRANSACTION_POINT_PREVIOUS);
        self.notify_before(&event)?;
        self.history.undo()?;
        self.notify_changed(&event);
        Ok(())
    }

    /// Redo after observers accept the move; example: `session.redo()?;`.
    pub fn redo(&mut self) -> Result<(), SessionError> {
        let label = self.history.redo_label()?;
        let event = self.payload(EVENT_TRANSACTION_REDONE, label, TRANSACTION_POINT_NEXT);
        self.notify_before(&event)?;
        self.history.redo()?;
        self.notify_changed(&event);
        Ok(())
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
