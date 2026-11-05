// SPDX-License-Identifier: GPL-2.0-only
use super::*;
/// A complete clipboard shutdown outcome; absence of a manager is a normal desktop state.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum NativeClipboardHandoff {
    NotOwned,
    NoManager,
    Saved,
    Declined,
}
impl NativeClipboardEndpoint {
    /// Offer current owned text to the desktop manager before releasing the endpoint.
    /// Example: `clipboard.handoff_on_shutdown()?` after the GUI stops accepting input.
    pub fn handoff_on_shutdown(&self) -> Result<NativeClipboardHandoff, String> {
        if self.0.selection_owner(self.0.atoms.clipboard)? != self.0.window {
            return Ok(NativeClipboardHandoff::NotOwned);
        }
        if self.0.selection_owner(self.0.atoms.clipboard_manager)? == 0 {
            return Ok(NativeClipboardHandoff::NoManager);
        }
        let deadline = self.0.operation_deadline()?;
        self.0.with_requestor(|requestor| {
            let timestamp = self.0.server_timestamp()?;
            self.0.begin_handoff(requestor, timestamp)?;
            self.0.await_handoff(requestor, timestamp, deadline)
        })
    }
}
impl SelectionEndpoint {
    fn begin_handoff(&self, requestor: Window, timestamp: Timestamp) -> Result<(), String> {
        self.write_words(
            requestor,
            self.atoms.handoff,
            AtomEnum::ATOM.into(),
            &[self.atoms.utf8],
        )?;
        self.request_conversion(
            requestor,
            self.atoms.clipboard_manager,
            self.atoms.save_targets,
            self.atoms.handoff,
            timestamp,
        )
    }
    fn await_handoff(
        &self,
        requestor: Window,
        timestamp: Timestamp,
        deadline: Instant,
    ) -> Result<NativeClipboardHandoff, String> {
        loop {
            super::receive::require_live_deadline(deadline)?;
            self.expire_transfers()?;
            let Some(event) = self.next_event()? else {
                let expiry = self
                    .outbound
                    .borrow()
                    .deadline()
                    .unwrap_or(deadline)
                    .min(deadline);
                properties::wait_selection(&self.connection, expiry)?;
                continue;
            };
            if let Event::SelectionNotify(notification) = &event
                && self.is_handoff_notification(notification, requestor, timestamp)
            {
                return classify_handoff_property(notification.property, self.atoms.handoff);
            }
            self.service_event(event)?;
        }
    }
    fn is_handoff_notification(
        &self,
        notification: &SelectionNotifyEvent,
        requestor: Window,
        timestamp: Timestamp,
    ) -> bool {
        notification.requestor == requestor
            && notification.selection == self.atoms.clipboard_manager
            && notification.target == self.atoms.save_targets
            && notification.time == timestamp
    }
}
fn classify_handoff_property(
    property: Atom,
    expected: Atom,
) -> Result<NativeClipboardHandoff, String> {
    match property {
        0 => Ok(NativeClipboardHandoff::Declined),
        property if property == expected => Ok(NativeClipboardHandoff::Saved),
        property => Err(format!(
            "clipboard handoff property {property}: expected {expected} or refusal 0"
        )),
    }
}
