// SPDX-License-Identifier: GPL-2.0-only
//! Content lifecycle fixtures must not assume the document contains a width parameter.
use super::*;

#[derive(Default)]
pub(super) struct ContentObserver;

impl TransactionObserver for ContentObserver {
    fn before(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) -> Result<(), String> {
        Ok(())
    }

    fn after(&mut self, _: &TransactionEventPayload, _: &DocumentSnapshot) {}

    fn changed(&mut self, _: u64, _: u64) {}
}
