// SPDX-License-Identifier: GPL-2.0-only
//! Access the displayed scene's status.
use super::NativeHeadWindow;

impl NativeHeadWindow {
    /// Inspect the displayed generation and declines; example: `window.scene_status()`.
    pub fn scene_status(&self) -> Option<&crate::HeadSceneStatus> {
        self.scene_status.as_ref()
    }
}
