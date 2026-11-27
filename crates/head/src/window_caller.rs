// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadWindow, ScriptDocumentOwner};
use oblikovati_api::{Error, client::Caller};
use std::{
    cell::{Cell, RefCell},
    collections::BTreeMap,
};

/// Combine document calls with owner-thread viewport automation.
pub(crate) struct WindowCaller<'a, D> {
    owner: RefCell<&'a mut D>,
    window: RefCell<&'a mut dyn HeadWindow>,
    window_changed: Cell<bool>,
    camera_slots: BTreeMap<u64, u64>,
}

impl<'a, D> WindowCaller<'a, D> {
    /// Bind one document caller and its live window; example: `WindowCaller::new(owner, window, slots)`.
    pub(crate) fn new(
        owner: &'a mut D,
        window: &'a mut dyn HeadWindow,
        camera_slots: BTreeMap<u64, u64>,
    ) -> Self {
        Self {
            owner: RefCell::new(owner),
            window: RefCell::new(window),
            window_changed: Cell::new(false),
            camera_slots,
        }
    }

    pub(crate) fn window_changed(&self) -> bool {
        self.window_changed.get()
    }
}

impl<D: ScriptDocumentOwner> Caller for WindowCaller<'_, D> {
    fn call(&self, method: &str, request: &[u8]) -> Result<Vec<u8>, Error> {
        let mut window = self.window.borrow_mut();
        let mut owner = self.owner.borrow_mut();
        if method == oblikovati_api::wire::METHOD_COMMANDS_EXECUTE {
            let value = crate::frames::execute_command(&mut **window, &mut **owner, request)?;
            self.window_changed.set(true);
            return Ok(value);
        }
        let model = owner.script_host();
        use oblikovati_api::wire::{METHOD_VIEW_GET_CAMERA, METHOD_VIEW_SET_CAMERA};
        if matches!(method, METHOD_VIEW_GET_CAMERA | METHOD_VIEW_SET_CAMERA) {
            let reply = crate::camera_automation::apply(
                &model,
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
            if reply.is_ok() && updates_window(method) {
                self.window_changed.set(true);
            }
            return reply;
        }
        model.call(method, request)
    }
}

fn updates_window(method: &str) -> bool {
    matches!(
        method,
        oblikovati_api::wire::METHOD_VIEWPORT_CAPTURE
            | oblikovati_api::wire::METHOD_VIEWPORT_CAPTURE_WINDOW
            | oblikovati_api::wire::METHOD_VIEWPORT_SET_NORMAL_DEBUG
            | oblikovati_api::wire::METHOD_VIEWPORT_SET_MESH_COLORS
    )
}
