// SPDX-License-Identifier: GPL-2.0-only
//! Non-blocking worker wakeup token for WebAssembly browser loops.
use std::sync::{
    Arc,
    atomic::{AtomicBool, Ordering},
};

/// A copied readiness result; DOM events and a worker signal can be ready together.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct ViewportReady {
    pub native_events: bool,
    pub worker_wake: bool,
}

/// Coalesced atomic wakeup token safe to share with WASM compute workers.
#[derive(Clone)]
pub struct ViewportWake {
    pending: Arc<AtomicBool>,
}

impl ViewportWake {
    /// Create a browser-safe wakeup signal; example: `let wake = ViewportWake::new()?;`.
    pub fn new() -> Result<Self, String> {
        Ok(Self {
            pending: Arc::new(AtomicBool::new(false)),
        })
    }

    /// Record a pending wakeup without blocking; example: `wake.wake()?;`.
    pub fn wake(&self) -> Result<(), String> {
        self.pending.store(true, Ordering::Release);
        Ok(())
    }

    pub(crate) fn consume(&self) -> Result<bool, String> {
        Ok(self.pending.swap(false, Ordering::AcqRel))
    }
}
