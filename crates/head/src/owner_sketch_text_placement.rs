// SPDX-License-Identifier: GPL-2.0-only
//! Start Format Text only after the canvas placement gesture completes.
use super::{GuiSketchTextDraft, HeadWindow, SketchTextTool, TextPlacementDraft};
use crate::owner::sketch_picking::ordinary_click;
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput};

impl SketchTextTool {
    /// Finish a point or drag placement with the release paired to its canvas press.
    /// For example, a drag from `[1050, 710]` uses that press as the insertion anchor.
    pub(super) fn place_on_release(
        &mut self,
        event: ViewportEvent,
        window: &dyn HeadWindow,
    ) -> Result<bool, String> {
        if window.navigation_active()
            || matches!(
                event,
                ViewportEvent::Focus(false) | ViewportEvent::Mapped(false)
            )
        {
            self.pressed_anchor = None;
            return Ok(false);
        }
        if self.placement.is_some() {
            return Ok(false);
        }
        if let Some((pixel, _)) = ordinary_click(event) {
            self.pressed_anchor = None;
            self.pressed_anchor = window.sketch_point(pixel, self.edit.plane)?;
            return Ok(self.pressed_anchor.is_some());
        }
        Ok(self.finish_pressed_placement(event))
    }

    fn finish_pressed_placement(&mut self, event: ViewportEvent) -> bool {
        let ViewportEvent::Input(ViewportInput::Button {
            button: ViewportButton::Left,
            pressed: false,
            modifiers,
            ..
        }) = event
        else {
            return false;
        };
        let Some(anchor) = self
            .pressed_anchor
            .take()
            .filter(|_| !modifiers.alt && !modifiers.super_key)
        else {
            return false;
        };
        self.placement = Some(TextPlacementDraft {
            anchor,
            fields: GuiSketchTextDraft::default(),
            existing: None,
            editor_fields: Default::default(),
        });
        true
    }
}
