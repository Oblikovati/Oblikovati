// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl<W: DispatchWakeup> SharedQueue<W> {
    pub(super) fn lock(&self, method: &str) -> Result<MutexGuard<'_, QueueStatus>, Error> {
        self.status.lock().map_err(|_| {
            Error(format!(
                "dispatch method {method:?}: expected an unpoisoned admission gate"
            ))
        })
    }
    pub(super) fn wait_capacity<'a>(
        &self,
        status: MutexGuard<'a, QueueStatus>,
        deadline: Instant,
        method: &str,
    ) -> Result<MutexGuard<'a, QueueStatus>, Error> {
        let remaining = deadline.saturating_duration_since(Instant::now());
        let (status, waited) = self
            .capacity_available
            .wait_timeout(status, remaining)
            .map_err(|_| {
                Error(format!(
                    "dispatch method {method:?}: expected an unpoisoned capacity wait"
                ))
            })?;
        if waited.timed_out() {
            return Err(self.timeout_error(method, false));
        }
        Ok(status)
    }
    pub(super) fn timeout_error(&self, method: &str, accepted: bool) -> Error {
        let admission = if accepted {
            "accepted request may still execute; do not retry automatically"
        } else {
            "request was not accepted"
        };
        Error(format!(
            "dispatch method {method:?} timed out: expected an owner reply within {:?}; {admission}",
            self.timeout
        ))
    }
    pub(super) fn closed_error(&self, method: &str) -> Error {
        Error(format!(
            "dispatch method {method:?}: dispatcher closed; expected a live owner"
        ))
    }
    pub(super) fn close(&self) {
        // Teardown must release waiters even after gate poisoning; this lock contains no model state.
        let mut status = self
            .status
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner);
        status.closed = true;
        status.requests.clear();
        self.capacity_available.notify_all();
    }
}
