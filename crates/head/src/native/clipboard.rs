// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_viewport::{NativeClipboardEndpoint, NativeClipboardOptions, ViewportClipboard};
use std::{cell::RefCell, rc::Rc, time::Duration};
// Owner policy bounds UI callback memory, pending clients, and external waits independently.
pub(super) const CLIPBOARD_BYTES: usize = 1024 * 1024;
pub(super) struct NativeClipboardChannel {
    endpoint: Result<NativeClipboardEndpoint, String>,
    issue: RefCell<Option<String>>,
}
impl NativeClipboardChannel {
    pub(super) fn open() -> Rc<Self> {
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
            endpoint,
        })
    }
    pub(super) fn endpoint(&self) -> Option<&NativeClipboardEndpoint> {
        self.endpoint.as_ref().ok()
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
        perform: impl FnOnce(&NativeClipboardEndpoint) -> Result<T, String>,
    ) -> Result<T, String> {
        let result = self
            .endpoint
            .as_ref()
            .map_err(Clone::clone)
            .and_then(perform);
        *self.issue.borrow_mut() = result.as_ref().err().cloned();
        result
    }
}
impl ViewportClipboard for NativeClipboardChannel {
    fn read_text(&self) -> Result<Option<String>, String> {
        self.operation(ViewportClipboard::read_text)
    }
    fn write_text(&self, text: &str) -> Result<(), String> {
        self.operation(|endpoint| endpoint.write_text(text))
    }
}
#[cfg(test)]
#[path = "../../tests/native_clipboard_channel.rs"]
mod tests;
