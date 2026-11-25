// SPDX-License-Identifier: GPL-2.0-only
//! What a window does each frame on its own: follow the selection, and apply navigation input.
use super::*;
impl NativeHeadWindow {
    pub(super) fn navigate_view_in_document(
        &mut self,
        command: oblikovati_viewport::ViewportNavigation,
    ) -> Result<bool, String> {
        self.orientation_transition = None;
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
                self.orientation_transition = None;
                changed |= self.restore_previous_scene_view()?;
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

// The trait methods share one native navigation owner.
macro_rules! native_view_inputs {
    () => {
        fn start_orientation_transition(
            &mut self,
            camera: oblikovati_viewport::ViewportCamera,
            finish: Option<oblikovati_viewport::ViewportNavigation>,
        ) -> Result<bool, String> {
            self.schedule_orientation(camera, finish)
        }
        fn tick_orientation_transition(
            &mut self,
        ) -> Result<(bool, Option<std::time::Duration>), String> {
            self.advance_orientation()
        }
        fn process_view_input(&mut self, event: &ViewportEvent) -> Result<bool, String> {
            self.apply_navigation_inputs(&[*event])
        }
        fn sketch_point(
            &self,
            pixel: [i32; 2],
            plane: oblikovati_model::sketch::SketchPlane,
        ) -> Result<Option<[f64; 2]>, String> {
            NativeHeadWindow::sketch_point(self, pixel, plane)
        }
        fn navigation_active(&self) -> bool {
            self.navigation.captures_pointer()
        }
        fn batches_navigation_motion(&self) -> bool {
            self.navigation.dragging()
        }
        fn allows_ready_sketch_motion(&self, pixel: [i32; 2]) -> bool {
            !self.navigation.captures_pointer()
                && self
                    .gui
                    .as_ref()
                    .is_some_and(|gui| gui.allows_ready_sketch_motion(pixel))
        }
    };
}
