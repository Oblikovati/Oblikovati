// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl HeadWindow for ScriptedGuiWindow {
    fn window_capture_extent(&mut self) -> Result<Option<[u32; 2]>, String> {
        self.native.window_capture_extent()
    }
    fn orientation_fit_frame(&mut self) -> Result<crate::HeadOrientationFitFrame, String> {
        self.native.orientation_fit_frame()
    }
    fn gui_parameters_open(&self) -> bool {
        self.native.gui_parameters_open()
    }
    fn set_gui_parameter_snapshot(
        &mut self,
        snapshot: Option<oblikovati_gui::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        self.native.set_gui_parameter_snapshot(snapshot)
    }
    fn show_gui_parameter_issue(&mut self, issue: String) -> Result<bool, String> {
        self.native.show_gui_parameter_issue(issue)
    }
    fn acknowledge_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
        result: Result<(), String>,
    ) -> Result<(), String> {
        self.native.acknowledge_gui_parameter(action, result)
    }
    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        self.native.set_sketch_preview(preview)
    }
    fn set_view_slot(&mut self, slot: u64) -> Result<bool, String> {
        self.native.set_view_slot(slot)
    }
    fn sketch_pixel(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        point: [f64; 2],
    ) -> Result<[i32; 2], String> {
        HeadWindow::sketch_pixel(&self.native, plane, point)
    }
    fn model_pixel(&self, point: [f64; 3]) -> Result<[i32; 2], String> {
        self.native.model_pixel(point)
    }
    fn view_camera(&self) -> Result<oblikovati_viewport::ViewportCamera, String> {
        self.native.view_camera()
    }
    fn view_line(&self, pixel: [i32; 2]) -> Result<[[f64; 3]; 2], String> {
        self.native.view_line(pixel)
    }
    fn navigate_view(
        &mut self,
        command: oblikovati_viewport::ViewportNavigation,
    ) -> Result<bool, String> {
        self.native.navigate_view(command)
    }
    fn previous_view(&mut self) -> Result<bool, String> {
        self.native.previous_view()
    }
    fn mark_view(&mut self) -> Result<(), String> {
        self.native.mark_view()
    }
    fn restore_marked_view(&mut self) -> Result<bool, String> {
        self.native.restore_marked_view()
    }
    fn view_projection(&self) -> oblikovati_gui::GuiViewProjection {
        self.native.view_projection()
    }
    fn set_view_projection(&mut self, mode: oblikovati_gui::GuiViewProjection) {
        self.native.set_view_projection(mode);
    }
    fn set_view_camera(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<bool, String> {
        self.native.set_view_camera(camera)
    }
    fn navigation_active(&self) -> bool {
        self.native.navigation_active()
    }
    fn sketch_point(
        &self,
        pixel: [i32; 2],
        plane: oblikovati_model::sketch::SketchPlane,
    ) -> Result<Option<[f64; 2]>, String> {
        self.native.sketch_point(pixel, plane)
    }
    fn set_sketch_edit(&mut self, edit: Option<crate::HeadSketchEdit>) -> Result<bool, String> {
        self.native.set_sketch_edit(edit)
    }
    fn gui_frame(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        let mut current = commands.clone();
        if let Some(entry) = &self.template_material_override {
            current
                .template_materials
                .retain(|item| item.template != entry.template);
            current.template_materials.push(entry.clone());
        }
        if let Some(cause) = &self.template_size_issue_override {
            current.template_size_issue = Some(cause.clone());
        }
        let result = self.native.gui_frame(event, &current)?;
        self.gui_actions.extend(result.actions.iter().cloned());
        Ok(result)
    }
    fn request_extrusion(&mut self) {
        self.native.request_extrusion();
    }
    fn reject_dimension_edit(&mut self, value: &oblikovati_gui::GuiDimensionValue, cause: &str) {
        self.native.reject_dimension_edit(value, cause);
    }
    fn reject_extrusion(&mut self, request: &oblikovati_gui::GuiExtrudePreview, cause: &str) {
        self.native.reject_extrusion(request, cause);
    }
    fn reject_offset_plane(
        &mut self,
        document: u64,
        base: oblikovati_gui::GuiOffsetPlaneBase,
        offset: &str,
        cause: &str,
    ) {
        self.native
            .reject_offset_plane(document, base, offset, cause);
    }
    fn capture_window_preview(&mut self) -> Result<bool, String> {
        self.native.capture_window_preview()
    }
    fn capture_isometric_preview(&mut self, document: u64) -> Result<bool, String> {
        self.native.capture_isometric_preview(document)
    }
    fn request_close(&mut self) -> Result<bool, String> {
        self.native.request_close()
    }
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        let mut events = self.native.events(limit)?;
        events.append(&mut self.scripted);
        Ok(events)
    }
    fn process_view_input(&mut self, event: &ViewportEvent) -> Result<bool, String> {
        self.passed.push(*event);
        self.native.process_view_input(event)
    }
    fn set_datum_planes(&mut self, update: &crate::HeadDatumPlaneUpdate) -> Result<bool, String> {
        self.native.set_datum_planes(update)
    }
    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.scene_triangles = update.scene.triangles.clone();
        self.scene_colors = update
            .scene
            .triangles
            .iter()
            .map(|triangle| triangle.color)
            .collect();
        self.native.submit_scene(update)
    }
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.native.submit_sketch_scene(update)
    }
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        self.native.show_command_notice(notice)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        if !self.presentation_delay.is_zero() {
            std::thread::sleep(self.presentation_delay);
        }
        self.native.present()
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
