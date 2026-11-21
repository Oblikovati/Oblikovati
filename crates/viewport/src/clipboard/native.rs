// SPDX-License-Identifier: GPL-2.0-only
use std::{
    cell::{Cell, RefCell},
    rc::Rc,
    time::{Duration, Instant},
};
use x11rb::wrapper::ConnectionExt as _;
use x11rb::{
    connection::Connection,
    protocol::{Event, xproto::*},
    rust_connection::RustConnection,
};
mod offers;
mod properties;
mod readiness;
mod receive;
mod requestor;
mod send;
mod shutdown;
mod text;
use properties::{ClipboardAtoms, clipboard_failure};
pub use shutdown::NativeClipboardHandoff;
/// Explicit clipboard transfer limits; no modelling tolerance or renderer timing is used.
#[derive(Clone, Copy, Debug)]
pub struct NativeClipboardOptions {
    pub byte_limit: usize,
    pub target_limit: usize,
    pub chunk_limit: usize,
    pub transfer_limit: usize,
    pub transfer_timeout: Duration,
}
impl NativeClipboardOptions {
    fn validate(self) -> Result<Self, String> {
        if self.byte_limit == 0
            || self.transfer_limit == 0
            || self.chunk_limit == 0
            || self.target_limit == 0
            || self.target_limit.checked_mul(4).is_none()
            || self.transfer_timeout.is_zero()
        {
            return Err(format!(
                "clipboard options {self:?}: expected positive byte and target budgets with representable sizes and transfer deadline"
            ));
        }
        Ok(self)
    }
}
struct SelectionEndpoint {
    connection: RustConnection,
    window: Window,
    screen: usize,
    atoms: ClipboardAtoms,
    options: NativeClipboardOptions,
    snapshot: RefCell<Option<Rc<str>>>,
    outbound: RefCell<send::OutboundTransfers>,
    acquired: Cell<Timestamp>,
    pending_event: RefCell<Option<Event>>,
}
/// One owner-thread selection endpoint. Native handles and protocol events stay private.
/// ```compile_fail
/// fn require_send<T: Send>() {}
/// require_send::<oblikovati_viewport::NativeClipboardEndpoint>();
/// ```
#[derive(Clone)]
pub struct NativeClipboardEndpoint(Rc<SelectionEndpoint>);
impl NativeClipboardEndpoint {
    /// Open a hidden selection endpoint; example: `NativeClipboardEndpoint::connect(None, options)?`.
    pub fn connect(display: Option<&str>, options: NativeClipboardOptions) -> Result<Self, String> {
        let options = options.validate()?;
        let (connection, screen) = RustConnection::connect(display)
            .map_err(|cause| clipboard_failure("connect", cause))?;
        let window = create_selection_window(&connection, screen)?;
        let atoms = ClipboardAtoms::load(&connection)?;
        Ok(Self(Rc::new(SelectionEndpoint {
            connection,
            window,
            screen,
            atoms,
            options,
            snapshot: RefCell::new(None),
            outbound: RefCell::new(send::OutboundTransfers::default()),
            acquired: Cell::new(0),
            pending_event: RefCell::new(None),
        })))
    }
    /// Publish a validated text snapshot; example: `clipboard.publish_text("width = 6 cm")?`.
    pub fn publish_text(&self, text: &str) -> Result<(), String> {
        self.0.require_text(text)?;
        let timestamp = self.0.server_timestamp()?;
        self.0
            .connection
            .set_selection_owner(self.0.window, self.0.atoms.clipboard, timestamp)
            .map_err(|cause| clipboard_failure("ownership request", cause))?
            .check()
            .map_err(|cause| clipboard_failure("ownership reply", cause))?;
        self.0.require_ownership()?;
        *self.0.snapshot.borrow_mut() = Some(Rc::from(text));
        self.0.acquired.set(timestamp);
        Ok(())
    }
    /// Next pending send expiry; example: merge `clipboard.transfer_deadline()` into the owner wait.
    pub fn transfer_deadline(&self) -> Option<Instant> {
        self.0.outbound.borrow().deadline()
    }
    /// Serve bounded pending selection work; example: `clipboard.service_pending(32)?` in the owner loop.
    pub fn service_pending(&self, budget: usize) -> Result<usize, String> {
        self.0.expire_transfers()?;
        let mut consumed = 0;
        for _ in 0..budget {
            let Some(event) = self.0.next_event()? else {
                break;
            };
            self.0.service_event(event)?;
            consumed += 1;
        }
        self.0
            .connection
            .flush()
            .map_err(|cause| clipboard_failure("flush", cause))?;
        Ok(consumed)
    }
}
impl SelectionEndpoint {
    fn require_text(&self, text: &str) -> Result<(), String> {
        if text.len() > self.options.byte_limit || text.contains('\0') {
            return Err(format!(
                "clipboard text length {}, NUL={}: expected NUL-free text within {} bytes",
                text.len(),
                text.contains('\0'),
                self.options.byte_limit
            ));
        }
        Ok(())
    }
    fn require_ownership(&self) -> Result<(), String> {
        let owner = self.selection_owner(self.atoms.clipboard)?;
        if owner != self.window {
            return Err(format!(
                "clipboard owner {owner}: expected acquired window {}",
                self.window
            ));
        }
        Ok(())
    }
    fn selection_owner(&self, selection: Atom) -> Result<Window, String> {
        self.connection
            .get_selection_owner(selection)
            .map_err(|cause| clipboard_failure("owner query", cause))?
            .reply()
            .map(|reply| reply.owner)
            .map_err(|cause| clipboard_failure("owner reply", cause))
    }
    fn next_event(&self) -> Result<Option<Event>, String> {
        if let Some(event) = self.pending_event.borrow_mut().take() {
            return Ok(Some(event));
        }
        self.connection
            .poll_for_event()
            .map_err(|cause| clipboard_failure("event poll", cause))
    }
    fn service_event(&self, event: Event) -> Result<(), String> {
        match event {
            Event::SelectionRequest(request) => self.reply_selection(request),
            Event::PropertyNotify(property) if property.state == Property::DELETE => {
                self.send_next_chunk(property)
            }
            Event::DestroyNotify(destroyed) => {
                self.forget_requestor(destroyed.window);
                Ok(())
            }
            Event::SelectionClear(clear) if clear.selection == self.atoms.clipboard => {
                self.snapshot.borrow_mut().take();
                Ok(())
            }
            _ => Ok(()),
        }
    }
    fn server_timestamp(&self) -> Result<Timestamp, String> {
        self.connection
            .change_property8(
                PropMode::REPLACE,
                self.window,
                self.atoms.clock,
                AtomEnum::STRING,
                &[],
            )
            .map_err(|cause| clipboard_failure("clock request", cause))?
            .check()
            .map_err(|cause| clipboard_failure("clock reply", cause))?;
        let deadline = self.operation_deadline()?;
        self.await_timestamp(deadline)
    }
    fn await_timestamp(&self, deadline: Instant) -> Result<Timestamp, String> {
        loop {
            if Instant::now() >= deadline {
                return Err("clipboard clock deadline expired: expected a server timestamp".into());
            }
            if let Some(event) = self.next_event()? {
                if let Event::PropertyNotify(property) = &event
                    && property.window == self.window
                    && property.atom == self.atoms.clock
                {
                    return Ok(property.time);
                }
                self.service_event(event)?;
                continue;
            }
            properties::wait_selection(&self.connection, deadline)?;
        }
    }
    fn reply_selection(&self, request: SelectionRequestEvent) -> Result<(), String> {
        let property = if request.property == 0 {
            request.target
        } else {
            request.property
        };
        let operation = self.write_selection_property(request, property);
        let accepted = matches!(operation, Ok(true));
        let notification = self.notify_selection(request, if accepted { property } else { 0 });
        match (operation, notification) {
            (Ok(_), reply) => reply,
            (Err(cause), Ok(())) => Err(cause),
            (Err(cause), Err(reply)) => Err(format!("{cause}; selection reply: {reply}")),
        }
    }
    fn notify_selection(
        &self,
        request: SelectionRequestEvent,
        property: Atom,
    ) -> Result<(), String> {
        let reply = SelectionNotifyEvent {
            response_type: SELECTION_NOTIFY_EVENT,
            sequence: 0,
            time: request.time,
            requestor: request.requestor,
            selection: request.selection,
            target: request.target,
            property,
        };
        self.connection
            .send_event(false, request.requestor, EventMask::NO_EVENT, reply)
            .map_err(|cause| clipboard_failure("selection notification", cause))?
            .check()
            .map_err(|cause| clipboard_failure("selection notification reply", cause))
    }
    fn write_words(
        &self,
        window: Window,
        property: Atom,
        kind: Atom,
        words: &[u32],
    ) -> Result<bool, String> {
        self.connection
            .change_property32(PropMode::REPLACE, window, property, kind, words)
            .map_err(|cause| clipboard_failure("word property", cause))?
            .check()
            .map_err(|cause| clipboard_failure("word property reply", cause))?;
        Ok(true)
    }
    fn write_bytes(
        &self,
        window: Window,
        property: Atom,
        kind: Atom,
        bytes: &[u8],
    ) -> Result<bool, String> {
        self.connection
            .change_property8(PropMode::REPLACE, window, property, kind, bytes)
            .map_err(|cause| clipboard_failure("text property", cause))?
            .check()
            .map_err(|cause| clipboard_failure("text property reply", cause))?;
        Ok(true)
    }
}
fn create_selection_window(connection: &RustConnection, screen: usize) -> Result<Window, String> {
    let window = connection
        .generate_id()
        .map_err(|cause| clipboard_failure("window ID", cause))?;
    let root = connection.setup().roots[screen].root;
    create_hidden_window(connection, root, window)?;
    Ok(window)
}
fn create_hidden_window(
    connection: &RustConnection,
    root: Window,
    window: Window,
) -> Result<(), String> {
    connection
        .create_window(
            0,
            window,
            root,
            0,
            0,
            1,
            1,
            0,
            WindowClass::INPUT_ONLY,
            0,
            &CreateWindowAux::new().event_mask(EventMask::PROPERTY_CHANGE),
        )
        .map_err(|cause| clipboard_failure("helper window", cause))?
        .check()
        .map_err(|cause| clipboard_failure("helper window reply", cause))
}

#[cfg(test)]
#[path = "../../tests/clipboard/native.rs"]
mod tests;

impl super::ViewportClipboard for NativeClipboardEndpoint {
    fn read_text(&self) -> Result<Option<String>, String> {
        self.0.read_current_text()
    }
    fn write_text(&self, text: &str) -> Result<(), String> {
        self.publish_text(text)
    }
}
