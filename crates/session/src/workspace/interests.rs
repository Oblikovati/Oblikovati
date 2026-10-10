// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::INTERESTED;

impl<O: TransactionObserver, W: WorkspaceObserver> WorkspaceSession<O, W> {
    pub(crate) fn notify_unavailable_interest_clients(&mut self, id: u64) {
        let entry = &self.entries[&id];
        let absent: Vec<String> = entry
            .session
            .snapshot()
            .interests
            .records()
            .iter()
            .filter(|record| record.interest_type == INTERESTED)
            .filter(|record| !self.observer.interest_client_present(&record.client_id))
            .map(|record| record.client_id.clone())
            .collect();
        if absent.is_empty() {
            return;
        }
        self.observer
            .unavailable_interest_clients(id, &entry.display_name, &absent);
    }
}
