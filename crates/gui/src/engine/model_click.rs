// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl GuiFrameEngine {
    /// Apply a viewport hit to the armed extrusion slot. The callback runs only while awaiting a pick.
    /// Example: `engine.resolve_extrusion_model_pick(|| Some((document, face)))`.
    pub fn resolve_extrusion_model_pick(
        &mut self,
        hit: impl FnOnce() -> Option<(u64, String)>,
    ) -> bool {
        if !self.extrude_dialog.awaits_reference_pick() {
            return false;
        }
        let Some((document, face)) = hit() else {
            return false;
        };
        self.extrude_dialog.pick_reference(document, face);
        true
    }

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

impl crate::ribbon::DocumentRibbon {
    pub(crate) fn cancel_unrouted_model_pick(&mut self, actions: &[GuiShellAction]) -> bool {
        if actions
            .iter()
            .any(|action| matches!(action, GuiShellAction::ClickModel { .. }))
        {
            return false;
        }
        if !self.panels.resolve_model_pick(false) {
            return false;
        }
        self.tips.dismiss();
        true
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn consumed_click_cannot_transfer_to_the_next_model_click() {
        let mut ribbon = crate::ribbon::DocumentRibbon::default();
        ribbon.panels.begin_model_pick();
        assert!(ribbon.cancel_unrouted_model_pick(&[]));
        assert!(!ribbon.panels.has_pending_model_pick());
        let click = GuiShellAction::ClickModel {
            document: 1,
            pixel: [7, 9],
            toggle: false,
            extrude_profile: None,
        };
        assert!(!ribbon.cancel_unrouted_model_pick(&[click]));
        assert!(!ribbon.panels.has_pending_model_pick());
    }
}
