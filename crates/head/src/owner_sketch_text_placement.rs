// SPDX-License-Identifier: GPL-2.0-only
//! Start Format Text only after the canvas placement gesture completes.
use super::{GuiSketchTextDraft, HeadWindow, SketchTextTool, TextPlacementDraft};
use crate::owner::sketch_picking::ordinary_click;
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput};
use std::sync::Arc;

const PREFERRED_TEXT_FAMILY: &str = "Tahoma";

impl SketchTextTool {
    pub(super) fn installed_text_families(&self) -> &Arc<[String]> {
        self.font_families.get_or_init(|| {
            oblikovati_martensite::planar_curve::sketch::available_sketch_font_families().into()
        })
    }

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
            fields: initial_text_format(self.installed_text_families()),
            existing: None,
            editor_fields: Default::default(),
        });
        true
    }
}

fn initial_text_format(installed: &[String]) -> GuiSketchTextDraft {
    // An unavailable named face would make the first typed preview fail.
    let family = installed
        .iter()
        .find(|name| name.eq_ignore_ascii_case(PREFERRED_TEXT_FAMILY))
        .cloned()
        .unwrap_or_default();
    GuiSketchTextDraft {
        family,
        ..Default::default()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn preferred_text_face_requires_an_installed_family() {
        let mut installed = vec!["DejaVu Sans".to_owned()];
        assert_eq!(
            initial_text_format(&installed),
            GuiSketchTextDraft::default()
        );
        installed.push("Tahoma".to_owned());
        assert_eq!(initial_text_format(&installed).family, "Tahoma");
    }
}
