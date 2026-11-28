// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl HeadWindow for ScriptedGuiWindow {
    fn synchronize_projection_extent(&mut self) -> Result<bool, String> {
        self.native.synchronize_projection_extent()
    }
    fn hovered_constraint_glyph(
        &self,
    ) -> Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        oblikovati_model::sketch::SketchConstraintId,
    )> {
        self.native.hovered_constraint_glyph()
    }

    fn hovered_constraint_glyphs(
        &self,
    ) -> Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        Vec<oblikovati_gui::GuiConstraintCandidate>,
    )> {
        self.native.hovered_constraint_glyphs()
    }

    fn window_action(&mut self, action: oblikovati_gui::GuiWindowAction) -> Result<bool, String> {
        self.native.window_action(action)
    }

    fn open_directory(&mut self, path: &std::path::Path) -> Result<(), String> {
        self.opened_directories.push(path.to_owned());
        Ok(())
    }

    fn set_tool_candidate(
        &mut self,
        hover: Option<&oblikovati_viewport::ViewportSketchHover>,
    ) -> Result<bool, String> {
        self.native.set_tool_candidate(hover)
    }
    fn set_tool_feedback(
        &mut self,
        feedback: Option<&oblikovati_viewport::ViewportSketchFeedback>,
    ) -> Result<bool, String> {
        self.native.set_tool_feedback(feedback)
    }
    fn validate_gui_constraint_settings(
        &self,
        action: &oblikovati_gui::GuiShellAction,
    ) -> Result<(), String> {
        self.native.validate_gui_constraint_settings(action)
    }
    fn acknowledge_gui_constraint_settings(
        &mut self,
        action: &oblikovati_gui::GuiShellAction,
        active_document: Option<u64>,
        result: Result<(), String>,
    ) -> Result<(), String> {
        self.native
            .acknowledge_gui_constraint_settings(action, active_document, result)
    }
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
    fn acknowledge_gui_document_units(
        &mut self,
        document: u64,
        result: Result<(), String>,
    ) -> Result<(), String> {
        self.native.acknowledge_gui_document_units(document, result)
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
    fn sketch_pixels(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        points: &[[f64; 2]],
    ) -> Result<Vec<[i32; 2]>, String> {
        self.native.sketch_pixels(plane, points)
    }
    fn sketch_pixel_results(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        points: &[[f64; 2]],
    ) -> Result<Vec<Result<[i32; 2], String>>, String> {
        self.native.sketch_pixel_results(plane, points)
    }
    fn sketch_pixel(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        point: [f64; 2],
    ) -> Result<[i32; 2], String> {
        HeadWindow::sketch_pixel(&self.native, plane, point)
    }
    fn presentation_bounds(&self) -> Option<[[f64; 3]; 2]> {
        HeadWindow::presentation_bounds(&self.native)
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
    fn set_view_camera_recorded(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<bool, String> {
        self.native.set_view_camera_recorded(camera)
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
    fn start_orientation_transition(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
        finish: Option<oblikovati_viewport::ViewportNavigation>,
    ) -> Result<bool, String> {
        self.native.start_orientation_transition(camera, finish)
    }
    fn tick_orientation_transition(&mut self) -> Result<(bool, Option<Duration>), String> {
        self.native.tick_orientation_transition()
    }
    fn navigation_active(&self) -> bool {
        self.native.navigation_active()
    }
    fn allows_ready_sketch_motion(&self, pixel: [i32; 2]) -> bool {
        self.native.allows_ready_sketch_motion(pixel)
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
        if let Some((document, path)) = &self.document_path_override
            && let Some(storage) = current.document_storage.get_mut(document)
        {
            storage.path = Some(path.clone());
        }
        let result = self.native.gui_frame(event, &current)?;
        if let Some(trace) = &mut self.orientation_trace {
            trace.axes = current.view_axes;
        }
        if self.observe_sketch_after_prompt
            && self.sketch_after_prompt.is_none()
            && self
                .native
                .gui
                .as_ref()
                .and_then(|gui| gui.frame_controls())
                .is_some_and(|(_, popup, _, _)| !popup)
        {
            self.sketch_after_prompt =
                Some(commands.authoring == oblikovati_gui::GuiAuthoringContext::Sketch);
        }
        self.gui_actions.extend(result.actions.iter().cloned());
        Ok(result)
    }
    fn has_pending_ribbon_model_pick(&self) -> bool {
        self.native.has_pending_ribbon_model_pick()
    }
    fn resolve_ribbon_model_pick(&mut self, face: bool) {
        self.native.resolve_ribbon_model_pick(face);
    }
    fn resolve_extrusion_model_pick(&mut self, hit: &mut dyn FnMut() -> Option<(u64, String)>) {
        self.native.resolve_extrusion_model_pick(hit);
    }
    fn request_extrusion(&mut self) {
        self.native.request_extrusion();
    }
    fn request_revolve(&mut self) {
        self.native.request_revolve();
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
    fn capture_window_preview(
        &mut self,
        state: &oblikovati_gui::GuiCommandState,
    ) -> Result<bool, String> {
        self.native.capture_window_preview(state)
    }
    fn capture_isometric_preview(&mut self, document: u64) -> Result<bool, String> {
        self.native.capture_isometric_preview(document)
    }
    fn request_application_close_confirmation(&mut self) -> Result<(), String> {
        self.confirmation_requests += 1;
        self.native.request_application_close_confirmation()
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
        let outcome = self.native.present()?;
        if outcome == HeadPresentation::Completed
            && let Some(trace) = &mut self.orientation_trace
        {
            let capture = self.native.capture_window()?;
            let camera = self.native.view_camera()?;
            let top = crate::view_orientation::view_axes(&camera).toward[1];
            let bounds = application_bdd::observe_orientation_solid(&capture);
            trace.presentations.push(PresentedOrientation {
                camera,
                axes: trace.axes,
                bounds,
            });
            if trace.intermediate.is_none()
                && bounds.is_some()
                && top > trace.initial_top_direction + 0.01
                && top < 0.99
            {
                trace.intermediate = Some(capture);
            }
        }
        Ok(outcome)
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        self.native.wait(timeout)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.native.finish()
    }
}
