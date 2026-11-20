// SPDX-License-Identifier: GPL-2.0-only
//! Head window operations and lifecycle.
use super::*;

impl HeadWindow for NativeHeadWindow {
    fn synchronize_projection_extent(&mut self) -> Result<bool, String> {
        self.view.synchronize_projection_extent()
    }
    fn hovered_constraint_glyph(
        &self,
    ) -> Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        oblikovati_model::sketch::SketchConstraintId,
    )> {
        self.glyph_target
    }

    fn hovered_constraint_glyphs(
        &self,
    ) -> Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        Vec<oblikovati_gui::GuiConstraintCandidate>,
    )> {
        self.glyph_candidates.clone()
    }

    fn capture_window(&self) -> Result<oblikovati_viewport::ViewportCapture, String> {
        self.view.capture()
    }
    fn queue_window_capture(&mut self, path: &std::path::Path) -> Result<[u32; 2], String> {
        let extent = self.view.drawable_extent()?;
        self.request_window_captures(vec![crate::WindowCaptureJob {
            path: path.into(),
            extent,
        }])?;
        self.wake.wake()?;
        Ok(extent)
    }
    fn inject_viewport_input(&mut self, event: ViewportEvent) -> Result<(), String> {
        const MAX_INJECTED: usize = 256;
        if self.injected_events.len() >= MAX_INJECTED {
            return Err(format!(
                "injected events {}: expected fewer than {MAX_INJECTED} pending events",
                self.injected_events.len()
            ));
        }
        self.injected_events.push_back(event);
        self.wake.wake()?;
        Ok(())
    }
    fn gui_parameters_open(&self) -> bool {
        self.gui.as_ref().is_some_and(|gui| gui.parameters_open())
    }
    fn set_gui_parameter_snapshot(
        &mut self,
        snapshot: Option<oblikovati_gui::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        match &mut self.gui {
            Some(gui) => gui.set_parameters(snapshot),
            None => Ok(false),
        }
    }
    fn show_gui_parameter_issue(&mut self, issue: String) -> Result<bool, String> {
        match &mut self.gui {
            Some(gui) => gui.report_parameter_issue(issue),
            None => Err(format!(
                "GUI parameter view issue {issue:?}: expected an initialized GUI"
            )),
        }
    }
    fn acknowledge_gui_parameter(
        &mut self,
        action: &oblikovati_gui::GuiParameterAction,
        result: Result<(), String>,
    ) -> Result<(), String> {
        if let Some(gui) = &mut self.gui {
            gui.acknowledge_parameter(action, result);
        }
        Ok(())
    }
    fn acknowledge_gui_document_units(
        &mut self,
        document: u64,
        result: Result<(), String>,
    ) -> Result<(), String> {
        if let Some(gui) = &mut self.gui {
            gui.acknowledge_document_units(document, result);
        }
        Ok(())
    }
    fn validate_gui_constraint_settings(
        &self,
        action: &oblikovati_gui::GuiShellAction,
    ) -> Result<(), String> {
        self.gui
            .as_ref()
            .ok_or("Constraint settings: expected an initialized GUI")?
            .validate_constraint_settings(action)
    }
    fn acknowledge_gui_constraint_settings(
        &mut self,
        action: &oblikovati_gui::GuiShellAction,
        active_document: Option<u64>,
        result: Result<(), String>,
    ) -> Result<(), String> {
        if let Some(gui) = &mut self.gui {
            gui.acknowledge_constraint_settings(action, active_document, result);
        }
        Ok(())
    }
    fn window_capture_extent(&mut self) -> Result<Option<[u32; 2]>, String> {
        self.view.drawable_extent().map(Some)
    }
    fn orientation_fit_frame(&mut self) -> Result<crate::HeadOrientationFitFrame, String> {
        let extent = self.view.drawable_extent()?;
        let area = self
            .gui
            .as_ref()
            .ok_or("selection fit area: expected an active GUI")?
            .document_area(extent)?;
        Ok(crate::HeadOrientationFitFrame {
            extent,
            area,
            bounds: self.view.fit_visible_bounds()?,
        })
    }
    fn window_capture_ready(&self) -> bool {
        self.capture_requests.is_empty()
    }
    fn request_window_captures(
        &mut self,
        requests: Vec<crate::WindowCaptureJob>,
    ) -> Result<(), String> {
        if !self.capture_requests.is_empty()
            || requests.len() > crate::window_capture::WINDOW_CAPTURE_QUEUE_CAPACITY
        {
            return Err(format!(
                "window captures pending={}, offered={}: expected an empty target and at most {} requests",
                self.capture_requests.len(),
                requests.len(),
                crate::window_capture::WINDOW_CAPTURE_QUEUE_CAPACITY
            ));
        }
        self.capture_requests = requests;
        Ok(())
    }
    fn set_tool_candidate(
        &mut self,
        hover: Option<&oblikovati_viewport::ViewportSketchHover>,
    ) -> Result<bool, String> {
        self.update_tool_hover(hover)
    }
    fn set_tool_feedback(
        &mut self,
        feedback: Option<&oblikovati_viewport::ViewportSketchFeedback>,
    ) -> Result<bool, String> {
        self.update_tool_feedback(feedback)
    }
    fn window_action(&mut self, action: oblikovati_gui::GuiWindowAction) -> Result<bool, String> {
        let action = match action {
            oblikovati_gui::GuiWindowAction::Minimize => {
                oblikovati_viewport::ViewportWindowAction::Minimize
            }
            oblikovati_gui::GuiWindowAction::ToggleMaximize => {
                oblikovati_viewport::ViewportWindowAction::ToggleMaximize
            }
            oblikovati_gui::GuiWindowAction::Move => {
                oblikovati_viewport::ViewportWindowAction::Move
            }
        };
        self.view.window_action(action)?;
        Ok(true)
    }
    fn gui_frame(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &oblikovati_gui::GuiCommandState,
    ) -> Result<crate::HeadGuiUpdate, String> {
        self.note_document_title(commands)?;
        self.preview_background = commands
            .color_scheme
            .as_ref()
            .map(|palette| palette.background());
        let changed = match &commands.color_scheme {
            Some(palette) => self.view.set_background(palette.background())?,
            None => false,
        };
        let sketch_changed = match &commands.theme {
            Some(theme) => self.view.set_sketch_palette(theme.sketch_palette())?,
            None => false,
        };
        let selection_changed = self.synchronize_sketch_highlights(commands)?;
        let mut window_commands = commands.clone();
        window_commands.window_maximized = self.view.window_maximized()?;
        let mut update = self.advance_native_gui(event, &window_commands)?;
        self.update_accessibility(event, &window_commands, &mut update)?;
        if let Some(hover) = &update.constraint_hover {
            update.changed |= self.update_glyph_hover(hover.as_ref(), commands.document)?;
        }
        update.changed |= changed || sketch_changed || selection_changed;
        Ok(update)
    }
    fn request_extrusion(&mut self) {
        if let Some(gui) = self.gui.as_mut() {
            gui.request_extrusion();
        }
    }
    fn has_pending_ribbon_model_pick(&self) -> bool {
        self.gui
            .as_ref()
            .is_some_and(|gui| gui.has_pending_ribbon_model_pick())
    }
    fn resolve_ribbon_model_pick(&mut self, face: bool) {
        if let Some(gui) = self.gui.as_mut() {
            gui.resolve_ribbon_model_pick(face);
        }
    }
    fn reject_dimension_edit(&mut self, value: &oblikovati_gui::GuiDimensionValue, cause: &str) {
        if let Some(gui) = self.gui.as_mut() {
            gui.reject_dimension_edit(value, cause);
        }
    }
    fn reject_extrusion(&mut self, request: &oblikovati_gui::GuiExtrudePreview, cause: &str) {
        if let Some(gui) = self.gui.as_mut() {
            gui.reject_extrusion(request, cause);
        }
    }
    fn reject_offset_plane(
        &mut self,
        document: u64,
        base: oblikovati_gui::GuiOffsetPlaneBase,
        offset: &str,
        cause: &str,
    ) {
        if let Some(gui) = self.gui.as_mut() {
            gui.reject_offset_plane(document, base, offset, cause);
        }
    }
    fn reject_extrusion_edit(
        &mut self,
        request: &oblikovati_gui::GuiExtrudeEditRequest,
        cause: &str,
    ) {
        if let Some(gui) = self.gui.as_mut() {
            gui.reject_extrusion_edit(request, cause);
        }
    }
    fn set_datum_planes(&mut self, update: &crate::HeadDatumPlaneUpdate) -> Result<bool, String> {
        let changed = self.view.set_datum_planes(&update.planes)?;
        let declined = self.datum_declines != update.declines;
        if declined {
            self.datum_declines.clone_from(&update.declines);
            self.refresh_title()?;
        }
        Ok(changed || declined)
    }
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.update_sketch_scene(update)
    }
    fn set_sketch_edit(&mut self, edit: Option<crate::HeadSketchEdit>) -> Result<bool, String> {
        self.synchronize_sketch_camera(edit)
    }
    fn set_view_slot(&mut self, slot: u64) -> Result<bool, String> {
        self.switch_view_slot(slot)
    }
    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        let changed = self.view.set_sketch_preview(preview)?;
        if preview.is_none()
            || !matches!(
                self.command_notice,
                Some(crate::DesktopCommandNotice::PreviewFailed { .. })
            )
        {
            return Ok(changed);
        }
        self.view.set_status_text("")?;
        self.command_notice = None;
        self.refresh_title()?;
        Ok(true)
    }
    native_view_inputs!();

    fn sketch_pixels(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        points: &[[f64; 2]],
    ) -> Result<Vec<[i32; 2]>, String> {
        self.view.sketch_pixels(plane, points)
    }
    fn sketch_pixel_results(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        points: &[[f64; 2]],
    ) -> Result<Vec<Result<[i32; 2], String>>, String> {
        self.view.sketch_pixel_results(plane, points)
    }
    fn sketch_pixel(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        point: [f64; 2],
    ) -> Result<[i32; 2], String> {
        NativeHeadWindow::sketch_pixel(self, plane, point)
    }
    fn presentation_bounds(&self) -> Option<[[f64; 3]; 2]> {
        self.presentation_bounds
    }
    fn model_pixel(&self, point: [f64; 3]) -> Result<[i32; 2], String> {
        self.view.model_pixel(point)
    }
    fn view_camera(&self) -> Result<oblikovati_viewport::ViewportCamera, String> {
        self.view.camera_state()
    }
    fn set_view_camera(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<bool, String> {
        self.orientation_transition = None;
        self.apply_orientation_camera(camera)
    }
    fn set_view_camera_recorded(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<bool, String> {
        self.orientation_transition = None;
        let changed = self.view.set_camera_recorded(camera)?;
        if changed {
            self.view
                .navigate(oblikovati_viewport::ViewportNavigation::FitDepth)?;
        }
        Ok(changed)
    }
    fn view_line(&self, pixel: [i32; 2]) -> Result<[[f64; 3]; 2], String> {
        self.view.view_line(pixel)
    }

    fn navigate_view(
        &mut self,
        command: oblikovati_viewport::ViewportNavigation,
    ) -> Result<bool, String> {
        self.navigate_view_in_document(command)
    }

    fn previous_view(&mut self) -> Result<bool, String> {
        self.orientation_transition = None;
        self.restore_previous_scene_view()
    }

    fn mark_view(&mut self) -> Result<(), String> {
        self.marked_view = Some(self.view.camera_state()?);
        Ok(())
    }

    fn restore_marked_view(&mut self) -> Result<bool, String> {
        let Some(camera) = self.marked_view else {
            return Ok(false);
        };
        self.orientation_transition = None;
        self.view.set_camera(camera)?;
        self.refresh_current_scene_depth()?;
        Ok(true)
    }

    fn view_projection(&self) -> oblikovati_gui::GuiViewProjection {
        self.view_projection
    }

    fn set_view_projection(&mut self, mode: oblikovati_gui::GuiViewProjection) {
        self.view_projection = mode;
    }

    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        if let (crate::DesktopCommandNotice::SavePathRequired { document }, Some(gui)) =
            (&notice, &mut self.gui)
        {
            gui.request_save_path(*document)?;
            self.command_notice = None;
            return self.refresh_title();
        }
        if self.gui.is_none() {
            self.view.set_status_text(&notice.message())?;
        }
        self.command_notice = (notice != crate::DesktopCommandNotice::Ready).then_some(notice);
        self.refresh_title()
    }

    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        self.submit_model_scene(update)
    }
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
        #[cfg(target_os = "linux")]
        self.clipboard.service_pending(limit);
        let queued = limit.min(self.injected_events.len());
        let mut events: Vec<_> = self.injected_events.drain(..queued).collect();
        if events.len() < limit {
            events.extend(self.view.poll_events(limit - events.len())?);
        }
        if std::mem::take(&mut self.exit_requested) {
            events.push(ViewportEvent::CloseRequested);
        }
        Ok(events)
    }
    fn capture_window_preview(&mut self) -> Result<bool, String> {
        let image = self.scene_preview_image()?;
        if let Some(gui) = &mut self.gui {
            gui.receive_window_preview(image);
        }
        Ok(true)
    }
    fn capture_isometric_preview(&mut self, document: u64) -> Result<bool, String> {
        self.publish_isometric_preview(document)
    }
    fn request_application_close_confirmation(&mut self) -> Result<(), String> {
        let gui = self.gui.as_mut().ok_or_else(|| {
            "application close confirmation: expected an initialized GUI".to_string()
        })?;
        gui.request_application_close();
        Ok(())
    }
    fn request_close(&mut self) -> Result<bool, String> {
        self.exit_requested = true;
        Ok(true)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        let presentation = self.view.present()?;
        self.refresh_title()?;
        Ok(match presentation {
            ViewportPresentation::Completed(capture) => {
                self.note_orientation_presented();
                if let Some(gui) = &mut self.gui {
                    gui.note_presented_frame(capture.extent, &capture.rgba);
                }
                self.publish_window_captures(&capture)?;
                HeadPresentation::Completed
            }
            ViewportPresentation::SurfaceChanged(_) => {
                if let Some(gui) = &mut self.gui {
                    gui.require_publication();
                }
                HeadPresentation::SurfaceChanged
            }
        })
    }
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String> {
        let windows = self.drawing_windows.targets();
        #[cfg(target_os = "linux")]
        let ready = match self.clipboard.endpoint() {
            Some(clipboard) => self
                .view
                .wait_events_for_windows_with_clipboard(&windows, &self.wake, clipboard, timeout),
            None => self
                .view
                .wait_events_for_windows(&windows, &self.wake, timeout),
        }?;
        #[cfg(target_os = "windows")]
        let ready = self
            .view
            .wait_events_for_windows(&windows, &self.wake, timeout)?;
        Ok(ready.worker_wake)
    }
    fn finish(&mut self) -> Result<(), String> {
        let workspace_error = match (&mut self.gui, &self.ribbon_settings_directory) {
            (Some(gui), Some(directory)) => gui
                .save_workspace_layout()
                .and_then(|bytes| preferences::NativeWorkspaceLayout::new(directory).save(&bytes))
                .err(),
            _ => None,
        };
        self.drawing_windows.close();
        self.gui.take();
        #[cfg(target_os = "windows")]
        self.accessibility.take();
        self.capture_requests.clear();
        #[cfg(target_os = "linux")]
        let clipboard = self.clipboard.handoff_on_shutdown();
        self.view.close();
        let mut errors = if self.validate_vulkan {
            self.runtime.validation_errors()?
        } else {
            Vec::new()
        };
        #[cfg(target_os = "linux")]
        errors.extend(clipboard.err());
        errors.extend(workspace_error);
        if errors.is_empty() {
            return Ok(());
        }
        Err(format!(
            "head shutdown {errors:?}: expected saved workspace settings, completed clipboard retirement, and no renderer errors"
        ))
    }
}
