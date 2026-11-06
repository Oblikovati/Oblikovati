// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{ViewportReady, ViewportWake};
use rustix::fd::{AsFd, BorrowedFd};
impl NativeClipboardEndpoint {
    /// Wait for clipboard work, transfer expiry, or a worker signal without a frame tick.
    /// Example: `clipboard.wait_events(&wake, None)?` before bounded `service_pending(32)`.
    pub fn wait_events(
        &self,
        wake: &ViewportWake,
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        self.wait_with_sources(wake, &[], timeout)
    }
    pub(crate) fn wait_with_sources(
        &self,
        wake: &ViewportWake,
        sources: &[BorrowedFd<'_>],
        timeout: Option<Duration>,
    ) -> Result<ViewportReady, String> {
        if self.0.buffer_pending_event()? {
            return Ok(ViewportReady {
                native_events: true,
                worker_wake: wake.consume()?,
            });
        }
        let expiry = self.transfer_deadline();
        let timeout = bounded_wait_timeout(timeout, expiry, Instant::now());
        let mut descriptors = sources.to_vec();
        descriptors.push(self.0.connection.stream().as_fd());
        let mut ready = wake.wait_sources(&descriptors, timeout)?;
        ready.native_events |= expiry.is_some_and(|deadline| deadline <= Instant::now());
        Ok(ready)
    }
}
impl SelectionEndpoint {
    fn buffer_pending_event(&self) -> Result<bool, String> {
        let mut pending = self.pending_event.borrow_mut();
        if pending.is_none() {
            *pending = self
                .connection
                .poll_for_event()
                .map_err(|cause| clipboard_failure("readiness poll", cause))?;
        }
        Ok(pending.is_some())
    }
}
fn bounded_wait_timeout(
    requested: Option<Duration>,
    expiry: Option<Instant>,
    now: Instant,
) -> Option<Duration> {
    match (
        requested,
        expiry.map(|deadline| deadline.saturating_duration_since(now)),
    ) {
        (Some(requested), Some(remaining)) => Some(requested.min(remaining)),
        (requested, remaining) => requested.or(remaining),
    }
}
#[cfg(test)]
#[path = "../../../tests/clipboard/readiness.rs"]
mod tests;
