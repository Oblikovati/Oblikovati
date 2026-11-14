// SPDX-License-Identifier: GPL-2.0-only
//! What a window does each frame on its own: follow the selection, and apply navigation input.
use super::*;
impl NativeHeadWindow {
    pub(super) fn navigate_view_in_document(
        &mut self,
        command: oblikovati_viewport::ViewportNavigation,
    ) -> Result<bool, String> {
        if matches!(
            command,
            oblikovati_viewport::ViewportNavigation::Fit { .. }
                | oblikovati_viewport::ViewportNavigation::Home { .. }
        ) && let Some(gui) = &self.gui
        {
            let area = gui.document_area(self.view.drawable_extent()?)?;
            return self.view.navigate_in_region(command, area);
        }
        self.view.navigate(command)
    }
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
                changed |= self.navigate_view_in_document(command)?;
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

pub(super) fn validate_window_extent(extent: [u32; 2]) -> Result<(), String> {
    if extent.contains(&0) {
        return Err(format!(
            "native window extent {extent:?}: expected two positive pixel dimensions"
        ));
    }
    Ok(())
}
