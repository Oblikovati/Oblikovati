// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl GuiFrameEngine {
    /// Check whether a model click must resolve a ribbon expansion; example: `engine.has_pending_ribbon_model_pick()`.
    pub fn has_pending_ribbon_model_pick(&self) -> bool {
        self.ribbon.panels.has_pending_model_pick()
    }

    /// Resolve a deferred ribbon click from the owner's model pick; example: `engine.resolve_ribbon_model_pick(true)`.
    pub fn resolve_ribbon_model_pick(&mut self, face: bool) {
        if self.ribbon.panels.resolve_model_pick(face) && !face {
            self.ribbon.tips.dismiss();
        }
    }
}
