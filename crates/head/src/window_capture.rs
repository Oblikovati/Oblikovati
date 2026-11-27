// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::{CaptureWindowArgs, CaptureWindowResult};
use oblikovati_automation::WindowCaptureAccess;
use std::{collections::VecDeque, path::PathBuf};
pub(crate) const WINDOW_CAPTURE_QUEUE_CAPACITY: usize = 16;

/// One accepted image request, pinned to the drawable size observed for its reply.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct WindowCaptureJob {
    pub path: PathBuf,
    pub extent: [u32; 2],
}
/// Bounded view work, independent of document history and retained while unmapped.
pub struct DesktopWindowCaptures {
    directory: PathBuf,
    namespace: u64,
    sequence: u64,
    extent: Option<[u32; 2]>,
    pending: VecDeque<WindowCaptureJob>,
}
impl DesktopWindowCaptures {
    /// Inject the default output directory and per-desktop namespace.
    /// Example: `DesktopWindowCaptures::new(temp_dir, process_id)`.
    pub fn new(directory: PathBuf, namespace: u64) -> Self {
        Self {
            directory,
            namespace,
            sequence: 0,
            extent: None,
            pending: VecDeque::new(),
        }
    }
    pub(crate) fn observe_extent(&mut self, extent: Option<[u32; 2]>) {
        self.extent = extent;
    }
    pub(crate) fn take_pending(&mut self) -> Vec<WindowCaptureJob> {
        self.pending.drain(..).collect()
    }
    fn admitted_extent(&self) -> Result<[u32; 2], String> {
        self.extent.filter(|extent| extent.iter().all(|value| *value > 0 && *value <= i32::MAX as u32))
            .ok_or_else(|| format!("window capture extent {:?}: expected an observed positive signed-32-bit drawable size", self.extent))
    }
    fn destination(&self, requested: &str, sequence: u64) -> Result<PathBuf, String> {
        if requested.contains('\0') {
            return Err(format!(
                "window capture path {requested:?}: expected a path without NUL characters"
            ));
        }
        Ok(if requested.is_empty() {
            self.directory.join(format!(
                "oblikovati-window-{}-{sequence}.png",
                self.namespace
            ))
        } else {
            PathBuf::from(requested)
        })
    }
}
impl WindowCaptureAccess for DesktopWindowCaptures {
    fn request_window_capture(
        &mut self,
        args: &CaptureWindowArgs,
    ) -> Result<CaptureWindowResult, String> {
        let extent = self.admitted_extent()?;
        let sequence = self.capture_sequence()?;
        let path = self.destination(&args.path, sequence)?;
        let text = path.to_str().ok_or_else(|| {
            format!("window capture path {path:?}: expected UTF-8 for the API reply")
        })?;
        let reply = CaptureWindowResult {
            path: text.into(),
            width: i64::from(extent[0]),
            height: i64::from(extent[1]),
        };
        self.pending.push_back(WindowCaptureJob { path, extent });
        self.sequence = sequence;
        Ok(reply)
    }
}
impl DesktopWindowCaptures {
    fn capture_sequence(&self) -> Result<u64, String> {
        if self.pending.len() >= WINDOW_CAPTURE_QUEUE_CAPACITY {
            return Err(format!(
                "window capture queue {}: expected fewer than {WINDOW_CAPTURE_QUEUE_CAPACITY} pending requests",
                self.pending.len()
            ));
        }
        self.sequence.checked_add(1).ok_or_else(|| {
            format!(
                "window capture sequence {}: expected an unused u64",
                self.sequence
            )
        })
    }
}

#[cfg(test)]
#[path = "../tests/window_capture.rs"]
mod tests;
