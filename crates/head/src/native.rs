// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadPresentation, HeadSummary, HeadWindow};
#[allow(unsafe_code)]
#[path = "native/addin_launch.rs"]
mod addin_launch;
#[path = "native_preferences.rs"]
mod preferences;
use oblikovati_viewport::{
    Viewport, ViewportEvent, ViewportPresentation, ViewportRuntime, ViewportScene, ViewportWake,
};
pub struct NativeHeadWindow {
    capture_files: Box<dyn crate::WindowCaptureFiles>,
    capture_requests: Vec<crate::WindowCaptureJob>,
    exit_requested: bool,
    runtime: ViewportRuntime,
    view: Viewport,
    view_projection: oblikovati_gui::GuiViewProjection,
    marked_view: Option<oblikovati_viewport::ViewportCamera>,
    wake: ViewportWake,
    scene_status: Option<crate::HeadSceneStatus>,
    current_scene: Option<ViewportScene>,
    preview_background: Option<oblikovati_viewport::ViewportBackground>,
    sketch_scene: Option<crate::HeadSketchSceneUpdate>,
    navigation: crate::navigation::DesktopNavigation,
    sketch_view: Option<(crate::HeadSketchEdit, oblikovati_viewport::ViewportCamera)>,
    /// The window of a document in front; each keeps its own view while another is shown.
    /// What the title bar names: the active document, with its window number when it has one.
    document_title: Option<String>,
    view_slot: u64,
    saved_views: std::collections::BTreeMap<
        u64,
        (
            oblikovati_viewport::ViewportCamera,
            Option<(crate::HeadSketchEdit, oblikovati_viewport::ViewportCamera)>,
        ),
    >,
    command_notice: Option<crate::DesktopCommandNotice>,
    gui: Option<gui_loop::NativeGui>,
    injected_events: std::collections::VecDeque<ViewportEvent>,
    ribbon_settings_directory: Option<std::path::PathBuf>,
    gui_epoch: std::time::Instant,
    clipboard: std::rc::Rc<clipboard::NativeClipboardChannel>,
    // Last so the GUI context drops before another test acquires the lease.
    #[cfg(test)]
    _context_lease: crate::test_context::TestContextLease,
}
mod open;
mod scene_status;
#[path = "native/view_inputs.rs"]
mod view_inputs;
use view_inputs::validate_window_extent;
impl HeadWindow for NativeHeadWindow {
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
        let mut update = self.advance_native_gui(event, commands)?;
        update.changed |= changed || sketch_changed || selection_changed;
        Ok(update)
    }
    fn request_extrusion(&mut self) {
        if let Some(gui) = self.gui.as_mut() {
            gui.request_extrusion();
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
    fn reject_extrusion_edit(
        &mut self,
        request: &oblikovati_gui::GuiExtrudeEditRequest,
        cause: &str,
    ) {
        if let Some(gui) = self.gui.as_mut() {
            gui.reject_extrusion_edit(request, cause);
        }
    }
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        self.view.set_sketch_display(&update.sketches)?;
        self.sketch_scene = Some(update);
        self.refresh_title()
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

    fn sketch_pixel(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        point: [f64; 2],
    ) -> Result<[i32; 2], String> {
        NativeHeadWindow::sketch_pixel(self, plane, point)
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
        self.view.set_camera(camera)?;
        self.view
            .navigate(oblikovati_viewport::ViewportNavigation::FitDepth)?;
        Ok(true)
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
        self.view.previous_view()
    }

    fn mark_view(&mut self) -> Result<(), String> {
        self.marked_view = Some(self.view.camera_state()?);
        Ok(())
    }

    fn restore_marked_view(&mut self) -> Result<bool, String> {
        let Some(camera) = self.marked_view else {
            return Ok(false);
        };
        self.view.set_camera(camera)?;
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
        self.view.submit(&update.scene)?;
        self.current_scene = Some(update.scene);
        self.scene_status = Some(update.status);
        self.refresh_title()
    }

    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String> {
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
    fn request_close(&mut self) -> Result<bool, String> {
        self.exit_requested = true;
        Ok(true)
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        let presentation = self.view.present()?;
        self.refresh_title()?;
        Ok(match presentation {
            ViewportPresentation::Completed(capture) => {
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
        let ready = match self.clipboard.endpoint() {
            Some(clipboard) => self
                .view
                .wait_events_with_clipboard(&self.wake, clipboard, timeout),
            None => self.view.wait_events(&self.wake, timeout),
        }?;
        Ok(ready.worker_wake)
    }
    fn finish(&mut self) -> Result<(), String> {
        self.gui.take();
        self.capture_requests.clear();
        let clipboard = self.clipboard.handoff_on_shutdown();
        self.view.close();
        let mut errors = self.runtime.validation_errors()?;
        if let Err(cause) = clipboard {
            errors.push(cause);
        }
        if errors.is_empty() {
            return Ok(());
        }
        Err(format!(
            "head shutdown {errors:?}: expected completed clipboard retirement and no renderer errors"
        ))
    }
}
mod capture;
mod clipboard;
mod owner;
use owner::{NativeDesktopOwner, NativeOwnerWake, native_owner_at, native_script_engine};
#[cfg(test)]
use owner::{NativeScriptDesktop, native_owner, native_script_owner};
mod run;
pub use run::{run_native, run_native_documents};
#[cfg(test)]
#[path = "../tests/native_wait.rs"]
mod tests;

#[cfg(test)]
#[path = "../tests/native_owner.rs"]
mod owner_tests;

#[cfg(test)]
#[path = "../tests/native_scene.rs"]
mod scene_tests;

#[cfg(test)]
#[path = "../tests/navigation/native.rs"]
mod navigation_tests;

#[cfg(test)]
#[path = "../tests/native_startup.rs"]
mod startup_tests;

mod title;

#[cfg(test)]
#[path = "../tests/native_save.rs"]
mod save_tests;

#[cfg(test)]
#[path = "../tests/native_history.rs"]
mod history_tests;

#[cfg(test)]
#[allow(unsafe_code)]
#[path = "../tests/native_services.rs"]
mod addin_services_tests;

mod projection;

mod sketch_edit;

mod sketch_display;

#[cfg(test)]
#[path = "../tests/native_sketch_display.rs"]
mod sketch_display_tests;

mod gui_composition;

#[cfg(test)]
#[path = "../tests/native_gui_composition.rs"]
mod gui_composition_tests;

mod gui_loop;
#[cfg(test)]
#[path = "../tests/native_gui_loop.rs"]
mod gui_loop_tests;
