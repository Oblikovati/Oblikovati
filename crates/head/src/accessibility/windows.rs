// SPDX-License-Identifier: GPL-2.0-only
//! Windows UI Automation adapter for the application bar.
use super::{
    actions::FrameActions,
    frame::{FrameControl, FrameSemantics},
};
use accesskit::{ActivationHandler, TreeUpdate};
use std::sync::{
    Arc, Mutex,
    mpsc::{self, Receiver},
};

struct Activation(Arc<Mutex<Option<FrameSemantics>>>);
impl ActivationHandler for Activation {
    fn request_initial_tree(&mut self) -> Option<TreeUpdate> {
        self.0.lock().ok()?.as_ref().map(FrameSemantics::tree)
    }
}

pub(crate) struct WindowsFrameAccessibility {
    adapter: accesskit_windows::SubclassingAdapter,
    snapshot: Arc<Mutex<Option<FrameSemantics>>>,
    requests: Receiver<FrameControl>,
}
impl WindowsFrameAccessibility {
    pub fn new(hwnd: isize, wake: impl Fn() + Send + 'static) -> Self {
        let snapshot = Arc::new(Mutex::new(None));
        let (sender, requests) = mpsc::channel();
        let adapter = accesskit_windows::SubclassingAdapter::new(
            accesskit_windows::HWND(hwnd as _),
            Activation(snapshot.clone()),
            FrameActions { sender, wake },
        );
        Self {
            adapter,
            snapshot,
            requests,
        }
    }
    pub fn update(&mut self, semantics: FrameSemantics) {
        if let Ok(mut snapshot) = self.snapshot.lock() {
            if snapshot.as_ref() == Some(&semantics) {
                return;
            }
            *snapshot = Some(semantics.clone());
        }
        if let Some(events) = self.adapter.update_if_active(|| semantics.tree()) {
            events.raise();
        }
    }
    pub fn next_request(&self) -> Option<FrameControl> {
        self.requests.try_recv().ok()
    }
}
