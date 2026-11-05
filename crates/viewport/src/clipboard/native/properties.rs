// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use rustix::event::{PollFd, PollFlags, Timespec, poll};
pub(super) struct ClipboardAtoms {
    pub clipboard: Atom,
    pub targets: Atom,
    pub timestamp: Atom,
    pub utf8: Atom,
    pub clock: Atom,
    pub receive: Atom,
    pub incr: Atom,
    pub multiple: Atom,
    pub atom_pair: Atom,
    pub clipboard_manager: Atom,
    pub save_targets: Atom,
    pub target_sizes: Atom,
    pub handoff: Atom,
    pub null: Atom,
}
impl ClipboardAtoms {
    pub(super) fn load(connection: &RustConnection) -> Result<Self, String> {
        Ok(Self {
            clipboard: intern(connection, b"CLIPBOARD")?,
            targets: intern(connection, b"TARGETS")?,
            timestamp: intern(connection, b"TIMESTAMP")?,
            utf8: intern(connection, b"UTF8_STRING")?,
            clock: intern(connection, b"_OBLIKOVATI_CLIPBOARD_CLOCK")?,
            receive: intern(connection, b"_OBLIKOVATI_CLIPBOARD_RECEIVE")?,
            incr: intern(connection, b"INCR")?,
            multiple: intern(connection, b"MULTIPLE")?,
            atom_pair: intern(connection, b"ATOM_PAIR")?,
            clipboard_manager: intern(connection, b"CLIPBOARD_MANAGER")?,
            save_targets: intern(connection, b"SAVE_TARGETS")?,
            target_sizes: intern(connection, b"TARGET_SIZES")?,
            handoff: intern(connection, b"_OBLIKOVATI_CLIPBOARD_HANDOFF")?,
            null: intern(connection, b"NULL")?,
        })
    }
}
fn intern(connection: &RustConnection, name: &[u8]) -> Result<Atom, String> {
    connection
        .intern_atom(false, name)
        .map_err(|cause| clipboard_failure("atom request", cause))?
        .reply()
        .map(|reply| reply.atom)
        .map_err(|cause| clipboard_failure("atom reply", cause))
}
impl SelectionEndpoint {
    pub(super) fn operation_deadline(&self) -> Result<Instant, String> {
        Instant::now()
            .checked_add(self.options.transfer_timeout)
            .ok_or_else(|| {
                format!(
                    "clipboard timeout {:?}: expected representable deadline",
                    self.options.transfer_timeout
                )
            })
    }
    pub(super) fn request_conversion(
        &self,
        requestor: Window,
        selection: Atom,
        target: Atom,
        property: Atom,
        timestamp: Timestamp,
    ) -> Result<(), String> {
        self.connection
            .convert_selection(requestor, selection, target, property, timestamp)
            .map_err(|cause| clipboard_failure("conversion", cause))?
            .check()
            .map_err(|cause| clipboard_failure("conversion reply", cause))
    }
    pub(super) fn read_native_property(
        &self,
        delete: bool,
        window: Window,
        property: Atom,
        kind: Atom,
        words: u32,
    ) -> Result<GetPropertyReply, String> {
        self.connection
            .get_property(delete, window, property, kind, 0, words)
            .map_err(|cause| clipboard_failure("property request", cause))?
            .reply()
            .map_err(|cause| clipboard_failure("property reply", cause))
    }
}
pub(super) fn clipboard_failure(operation: &str, cause: impl std::fmt::Display) -> String {
    format!("clipboard {operation} {cause}: expected a successful checked X11 operation")
}
pub(super) fn wait_selection(connection: &RustConnection, deadline: Instant) -> Result<(), String> {
    let duration = deadline
        .checked_duration_since(Instant::now())
        .ok_or("clipboard transfer deadline expired: expected a responsive selection peer")?;
    let timeout =
        Timespec::try_from(duration).map_err(|cause| clipboard_failure("deadline", cause))?;
    let mut descriptors = [PollFd::new(connection.stream(), PollFlags::IN)];
    poll(&mut descriptors, Some(&timeout))
        .map_err(|cause| clipboard_failure("readiness", cause))?;
    if descriptors[0]
        .revents()
        .intersects(PollFlags::ERR | PollFlags::HUP | PollFlags::NVAL)
    {
        return Err(format!(
            "clipboard readiness {:?}: expected a live connection",
            descriptors[0].revents()
        ));
    }
    Ok(())
}
