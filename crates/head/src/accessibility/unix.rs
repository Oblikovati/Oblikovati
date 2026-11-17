// SPDX-License-Identifier: GPL-2.0-only
//! AT-SPI adapter for application window semantics.
use super::{
    actions::FrameActions,
    frame::{FrameControl, FrameSemantics},
};
use accesskit::{ActivationHandler, DeactivationHandler, TreeUpdate};
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
struct Deactivation;
impl DeactivationHandler for Deactivation {
    fn deactivate_accessibility(&mut self) {}
}

pub(crate) struct UnixFrameAccessibility {
    adapter: accesskit_unix::Adapter,
    snapshot: Arc<Mutex<Option<FrameSemantics>>>,
    requests: Receiver<FrameControl>,
}
impl UnixFrameAccessibility {
    pub fn new(wake: impl Fn() + Send + 'static) -> Self {
        let snapshot = Arc::new(Mutex::new(None));
        let (sender, requests) = mpsc::channel();
        let adapter = accesskit_unix::Adapter::new(
            Activation(snapshot.clone()),
            FrameActions { sender, wake },
            Deactivation,
        );
        Self {
            adapter,
            snapshot,
            requests,
        }
    }
    pub fn update(&mut self, semantics: FrameSemantics, bounds: accesskit::Rect, focused: bool) {
        self.adapter.set_root_window_bounds(bounds, bounds);
        self.adapter.update_window_focus_state(focused);
        if let Ok(mut snapshot) = self.snapshot.lock() {
            if snapshot.as_ref() == Some(&semantics) {
                return;
            }
            *snapshot = Some(semantics.clone());
        }
        self.adapter.update_if_active(|| semantics.tree());
    }
    pub fn next_request(&self) -> Option<FrameControl> {
        self.requests.try_recv().ok()
    }
}
