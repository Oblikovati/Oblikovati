// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWindow;
use oblikovati_api::{Error, client::Caller};
use std::cell::RefCell;

/// Combine document calls with owner-thread viewport automation.
pub(crate) struct WindowCaller<'a, C> {
    model: C,
    window: RefCell<&'a mut dyn HeadWindow>,
}

impl<'a, C> WindowCaller<'a, C> {
    /// Bind one document caller and its live window; example: `WindowCaller::new(host, window)`.
    pub(crate) fn new(model: C, window: &'a mut dyn HeadWindow) -> Self {
        Self {
            model,
            window: RefCell::new(window),
        }
    }
}

impl<C: Caller> Caller for WindowCaller<'_, C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut window = self.window.borrow_mut();
        if let Some(reply) = crate::viewport_automation::apply(&mut **window, method, request) {
            return reply;
        }
        self.model.call(method, request)
    }
}
