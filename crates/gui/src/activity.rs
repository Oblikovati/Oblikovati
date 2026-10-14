// SPDX-License-Identifier: GPL-2.0-only
/// Copied reasons to schedule another GUI frame; the caller supplies time and the wake policy.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct GuiFrameActivity {
    pub queued_input: bool,
    pub pending_layout: bool,
    pub pending_navigation: bool,
    pub interaction: bool,
}
impl GuiFrameActivity {
    /// Keep deferred input, layout and interactive timers moving; example: `output.activity.needs_frame()`.
    pub fn needs_frame(self) -> bool {
        self.queued_input || self.pending_layout || self.pending_navigation || self.interaction
    }
}
#[cfg(test)]
#[path = "../tests/activity/unit.rs"]
mod tests;
