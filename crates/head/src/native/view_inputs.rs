// SPDX-License-Identifier: GPL-2.0-only
//! What a window does each frame on its own: follow the selection, and apply navigation input.
use super::*;
impl NativeHeadWindow {
    pub(super) fn synchronize_sketch_highlights(
        &mut self,
        commands: &oblikovati_gui::GuiCommandState,
    ) -> Result<bool, String> {
        let matching_document = self
            .sketch_scene
            .as_ref()
            .and_then(|scene| scene.source)
            .map(|(document, _)| document)
            == commands.document;
        let empty = oblikovati_viewport::ViewportSketchSelection::default();
        let selected = if matching_document {
            &commands.sketch_selection
        } else {
            &empty
        };
        self.view.set_sketch_selection(selected)
    }

    pub(super) fn apply_navigation_inputs(
        &mut self,
        inputs: &[ViewportEvent],
    ) -> Result<bool, String> {
        let mut changed = false;
        for event in inputs {
            let command = self.navigation.command(event)?;
            if previous_view_pressed(event) {
                changed |= self.view.previous_view()?;
                continue;
            }
            if let Some(command) = command {
                changed |= self.view.navigate(command)?;
            }
        }
        Ok(changed)
    }
}

fn previous_view_pressed(event: &ViewportEvent) -> bool {
    matches!(
        event,
        ViewportEvent::Input(oblikovati_viewport::ViewportInput::Key {
            key: oblikovati_viewport::ViewportKey::Previous,
            pressed: true
        })
    )
}
