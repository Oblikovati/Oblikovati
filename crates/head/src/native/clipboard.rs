// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::{
    NativeClipboardEndpoint, NativeClipboardOptions, Viewport, ViewportClipboard, WaylandClipboard,
};
use std::{cell::RefCell, rc::Rc, time::Duration};
// Owner policy bounds UI callback memory, pending clients, and external waits independently.
pub(super) const CLIPBOARD_BYTES: usize = 1024 * 1024;
pub(super) struct NativeClipboardChannel {
    backend: ClipboardBackend,
    issue: RefCell<Option<String>>,
}
enum ClipboardBackend {
    X11(Result<NativeClipboardEndpoint, String>),
    Wayland(WaylandClipboard),
}

impl NativeClipboardChannel {
    pub(super) fn open(view: &Viewport) -> Rc<Self> {
        if let Some(clipboard) = view.wayland_clipboard(CLIPBOARD_BYTES) {
            return Rc::new(Self {
                backend: ClipboardBackend::Wayland(clipboard),
                issue: RefCell::new(None),
            });
        }
        Self::with_endpoint(NativeClipboardEndpoint::connect(
            None,
            NativeClipboardOptions {
                byte_limit: CLIPBOARD_BYTES,
                target_limit: 64,
                chunk_limit: 64 * 1024,
                transfer_limit: 16,
                transfer_timeout: Duration::from_secs(3),
            },
        ))
    }
    pub(super) fn with_endpoint(endpoint: Result<NativeClipboardEndpoint, String>) -> Rc<Self> {
        Rc::new(Self {
            issue: RefCell::new(endpoint.as_ref().err().cloned()),
            backend: ClipboardBackend::X11(endpoint),
        })
    }
    pub(super) fn endpoint(&self) -> Option<&NativeClipboardEndpoint> {
        match &self.backend {
            ClipboardBackend::X11(endpoint) => endpoint.as_ref().ok(),
            ClipboardBackend::Wayland(_) => None,
        }
    }
    pub(super) fn issue(&self) -> Option<String> {
        self.issue.borrow().clone()
    }
    pub(super) fn service_pending(&self, budget: usize) {
        if let Some(endpoint) = self.endpoint()
            && let Err(cause) = endpoint.service_pending(budget)
        {
            *self.issue.borrow_mut() = Some(cause);
        }
    }
    pub(super) fn handoff_on_shutdown(&self) -> Result<(), String> {
        let Some(endpoint) = self.endpoint() else {
            return Ok(());
        };
        let result = endpoint.handoff_on_shutdown().map(|_| ());
        if let Err(cause) = &result {
            *self.issue.borrow_mut() = Some(cause.clone());
        }
        result
    }
    fn operation<T>(
        &self,
        perform: impl FnOnce(&dyn ViewportClipboard) -> Result<T, String>,
    ) -> Result<T, String> {
        let result = match &self.backend {
            ClipboardBackend::X11(endpoint) => endpoint
                .as_ref()
                .map_err(Clone::clone)
                .and_then(|clipboard| perform(clipboard)),
            ClipboardBackend::Wayland(clipboard) => perform(clipboard),
        };
        *self.issue.borrow_mut() = result.as_ref().err().cloned();
        result
    }
}
impl ViewportClipboard for NativeClipboardChannel {
    fn read_text(&self) -> Result<Option<String>, String> {
        self.operation(|clipboard| clipboard.read_text())
    }
    fn write_text(&self, text: &str) -> Result<(), String> {
        self.operation(|endpoint| endpoint.write_text(text))
    }
}
#[cfg(test)]
#[path = "../../tests/native_clipboard_channel.rs"]
mod tests;
