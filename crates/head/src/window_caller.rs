// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWindow;
use oblikovati_api::{Error, client::Caller};
use std::cell::{Cell, RefCell};

/// Combine document calls with owner-thread viewport automation.
pub(crate) struct WindowCaller<'a, C> {
    model: C,
    window: RefCell<&'a mut dyn HeadWindow>,
    capture_queued: Cell<bool>,
}

impl<'a, C> WindowCaller<'a, C> {
    /// Bind one document caller and its live window; example: `WindowCaller::new(host, window)`.
    pub(crate) fn new(model: C, window: &'a mut dyn HeadWindow) -> Self {
        Self {
            model,
            window: RefCell::new(window),
            capture_queued: Cell::new(false),
        }
    }

    pub(crate) fn capture_queued(&self) -> bool {
        self.capture_queued.get()
    }
}

impl<C: Caller> Caller for WindowCaller<'_, C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut window = self.window.borrow_mut();
        if let Some(reply) = crate::viewport_automation::apply(&mut **window, method, request) {
            self.capture_queued
                .set(reply.is_ok() && capture_method(method));
            return reply;
        }
        self.model.call(method, request)
    }
}

fn capture_method(method: &str) -> bool {
    matches!(
        method,
        oblikovati_api::wire::METHOD_VIEWPORT_CAPTURE
            | oblikovati_api::wire::METHOD_VIEWPORT_CAPTURE_WINDOW
    )
}
