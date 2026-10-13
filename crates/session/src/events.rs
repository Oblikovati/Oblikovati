// SPDX-License-Identifier: GPL-2.0-only

use crate::{DocumentSession, SessionError};
use oblikovati_api::{types::TransactionPoint, wire::TransactionEventPayload};
use oblikovati_model::history::DocumentSnapshot;

/// In-process policy and observations. An add-in transport can relay After payloads.
pub trait TransactionObserver {
    /// Refuse a move with a visible reason; example: `Err("external export is locked".into())`.
    fn before(
        &mut self,
        event: &TransactionEventPayload,
        snapshot: &DocumentSnapshot,
    ) -> Result<(), String>;
    /// Observe completed movement; example: append `event.clone()` to an outbound queue.
    fn after(&mut self, event: &TransactionEventPayload, snapshot: &DocumentSnapshot);
    /// Observe a published evaluation generation; example: schedule evaluation of `generation`.
    fn changed(&mut self, document: u64, generation: u64);
}

impl<O: TransactionObserver> DocumentSession<O> {
    pub(crate) fn payload(
        &self,
        kind: &str,
        label: &str,
        point: TransactionPoint,
    ) -> TransactionEventPayload {
        TransactionEventPayload {
            type_: kind.into(),
            document: self.document,
            label: label.into(),
            point,
        }
    }

    pub(crate) fn notify_before(
        &mut self,
        event: &TransactionEventPayload,
    ) -> Result<(), SessionError> {
        self.observer
            .before(event, self.history.snapshot())
            .map_err(SessionError::Veto)
    }

    pub(crate) fn notify_after(&mut self, event: &TransactionEventPayload) {
        self.sketch_continuity.refresh(self.history.snapshot());
        self.observer.after(event, self.history.snapshot());
    }

    pub(crate) fn notify_changed(&mut self, event: &TransactionEventPayload) {
        self.sketch_continuity.refresh(self.history.snapshot());
        self.observer
            .changed(self.document, self.history.generation());
        self.observer.after(event, self.history.snapshot());
    }
}
