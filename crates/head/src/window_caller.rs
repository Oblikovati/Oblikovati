// SPDX-License-Identifier: GPL-2.0-only
use crate::HeadWindow;
use oblikovati_api::{Error, client::Caller};
use std::{
    cell::{Cell, RefCell},
    collections::BTreeMap,
};

/// Combine document calls with owner-thread viewport automation.
pub(crate) struct WindowCaller<'a, C> {
    model: C,
    window: RefCell<&'a mut dyn HeadWindow>,
    window_changed: Cell<bool>,
    camera_slots: BTreeMap<u64, u64>,
}

impl<'a, C> WindowCaller<'a, C> {
    /// Bind one document caller and its live window; example: `WindowCaller::new(host, window, slots)`.
    pub(crate) fn new(
        model: C,
        window: &'a mut dyn HeadWindow,
        camera_slots: BTreeMap<u64, u64>,
    ) -> Self {
        Self {
            model,
            window: RefCell::new(window),
            window_changed: Cell::new(false),
            camera_slots,
        }
    }

    pub(crate) fn window_changed(&self) -> bool {
        self.window_changed.get()
    }
}

impl<C: Caller> Caller for WindowCaller<'_, C> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut window = self.window.borrow_mut();
        use oblikovati_api::wire::{METHOD_VIEW_GET_CAMERA, METHOD_VIEW_SET_CAMERA};
        if matches!(method, METHOD_VIEW_GET_CAMERA | METHOD_VIEW_SET_CAMERA) {
            let reply = crate::camera_automation::apply(
                &self.model,
                &mut **window,
                &self.camera_slots,
                method,
                request,
            );
            if reply.is_ok() && method == METHOD_VIEW_SET_CAMERA {
                self.window_changed.set(true);
            }
            return reply;
        }
        if let Some(reply) = crate::viewport_automation::apply(&mut **window, method, request) {
            if reply.is_ok() && capture_method(method) {
                self.window_changed.set(true);
            }
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
