// SPDX-License-Identifier: GPL-2.0-only
//! Native window boundary owned by the desktop frame loop.
use super::{HeadGuiUpdate, HeadPresentation};
use oblikovati_viewport::ViewportEvent;

/// The displayed scene and unobstructed area used only to fit an orientation view.
/// These presentation bounds must not be used for model or topology decisions.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct HeadOrientationFitFrame {
    pub extent: [u32; 2],
    pub area: [u32; 4],
    pub bounds: [[f64; 3]; 2],
}

/// The desktop owns this boundary; implementations contain all native I/O.
pub trait HeadWindow {
    /// Synchronize camera and drawable extents; example: `window.synchronize_projection_extent()?`.
    fn synchronize_projection_extent(&mut self) -> Result<bool, String> {
        Ok(false)
    }
    /// Dispatch an application-bar window request; example: `window.window_action(action)?`.
    fn window_action(&mut self, action: oblikovati_gui::GuiWindowAction) -> Result<bool, String> {
        Err(format!(
            "window action {action:?}: expected a native window"
        ))
    }

    /// Observe the drawable used for capture metadata; example: `window.window_capture_extent()?`.
    fn window_capture_extent(&mut self) -> Result<Option<[u32; 2]>, String> {
        Ok(None)
    }
    /// Admit another batch only after the previous batch completes.
    /// Example: `window.window_capture_ready()` before transferring owner requests.
    fn window_capture_ready(&self) -> bool {
        false
    }
    /// Retain requests until a completed frame; example: `window.request_window_captures(jobs)?`.
    fn request_window_captures(
        &mut self,
        requests: Vec<crate::WindowCaptureJob>,
    ) -> Result<(), String> {
        Err(format!(
            "window capture requests {requests:?}: expected a native completed-frame capture target"
        ))
    }
    /// Picture the active workspace for Save As; example: `window.capture_window_preview(&state)?`.
    fn capture_window_preview(
        &mut self,
        _state: &oblikovati_gui::GuiCommandState,
    ) -> Result<bool, String> {
        Err("window preview: expected a native window that presents frames".into())
    }
    /// Render a document's isometric Save As preview without changing its live view.
    /// Example: `window.capture_isometric_preview(document)?`.
    fn capture_isometric_preview(&mut self, document: u64) -> Result<bool, String> {
        Err(format!(
            "isometric preview for document {document}: expected a native offscreen renderer"
        ))
    }
    /// Ask the window to close through its own close policy; example: the File menu's Exit.
    fn request_close(&mut self) -> Result<bool, String> {
        Err("exit: expected a native window that can close".into())
    }
    /// Queue the application close confirmation; example: `window.request_application_close_confirmation()?`.
    fn request_application_close_confirmation(&mut self) -> Result<(), String> {
        Err("application close confirmation: expected a window with a GUI".into())
    }
    /// Capture the current composed window on the owner thread; example: `window.capture_window()?`.
    fn capture_window(&self) -> Result<oblikovati_viewport::ViewportCapture, String> {
        Err("window capture: expected a native capture-capable window".into())
    }

    /// Queue a capture after the next presented frame; example: `window.queue_window_capture(path)?`.
    fn queue_window_capture(&mut self, _: &std::path::Path) -> Result<[u32; 2], String> {
        Err("window capture: expected a native capture-capable window".into())
    }

    /// Inject one viewport event through the same path as native input; example: `window.inject_viewport_input(event)?`.
    fn inject_viewport_input(
        &mut self,
        _: oblikovati_viewport::ViewportEvent,
    ) -> Result<(), String> {
        Err("viewport input: expected a native input-capable window".into())
    }

    /// Request owner metadata only while visible; example: `window.gui_parameters_open()`.
    fn gui_parameters_open(&self) -> bool {
        false
    }
    /// Copy the current parameter view; example: `window.set_gui_parameter_snapshot(None)?` on close.
    fn set_gui_parameter_snapshot(
        &mut self,
        _: Option<oblikovati_gui::GuiParameterSnapshot>,
    ) -> Result<bool, String> {
        Ok(false)
    }
    /// Show a parameter view decline locally; example: `window.show_gui_parameter_issue(issue)?`.
    fn show_gui_parameter_issue(&mut self, issue: String) -> Result<bool, String> {
        Err(format!(
            "GUI parameter view issue {issue:?}: expected a window with parameter feedback"
        ))
    }
    /// Return one owner result to its widget; example: `window.acknowledge_gui_parameter(&action, result)?`.
    fn acknowledge_gui_parameter(
        &mut self,
        _: &oblikovati_gui::GuiParameterAction,
        _: Result<(), String>,
    ) -> Result<(), String> {
        Ok(())
    }

    /// Report a unit update result; example: `window.acknowledge_gui_document_units(1, Ok(()))?`.
    fn acknowledge_gui_document_units(
        &mut self,
        _: u64,
        _: Result<(), String>,
    ) -> Result<(), String> {
        Ok(())
    }

    /// Check a pending settings draft; example: `window.validate_gui_constraint_settings(action)?`.
    fn validate_gui_constraint_settings(
        &self,
        _: &oblikovati_gui::GuiShellAction,
    ) -> Result<(), String> {
        Ok(())
    }

    /// Return the document settings result; example: `window.acknowledge_gui_constraint_settings(action, Some(1), Ok(()))?`.
    fn acknowledge_gui_constraint_settings(
        &mut self,
        _: &oblikovati_gui::GuiShellAction,
        _: Option<u64>,
        _: Result<(), String>,
    ) -> Result<(), String> {
        Ok(())
    }

    /// Advance GUI work before routing one input; example: `window.gui_frame(Some(event), &commands)?`.
    fn gui_frame(
        &mut self,
        _: Option<&ViewportEvent>,
        _: &oblikovati_gui::GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        Ok(HeadGuiUpdate::default())
    }
    /// Read the current unambiguous constraint glyph under the pointer.
    /// Example: `window.hovered_constraint_glyph()` before a context request.
    fn hovered_constraint_glyph(
        &self,
    ) -> Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        oblikovati_model::sketch::SketchConstraintId,
    )> {
        None
    }
    /// Return the current glyph candidates; example: `window.hovered_constraint_glyphs()`.
    fn hovered_constraint_glyphs(
        &self,
    ) -> Option<(
        oblikovati_model::document_sketch::DocumentSketchId,
        Vec<oblikovati_gui::GuiConstraintCandidate>,
    )> {
        self.hovered_constraint_glyph().map(|(sketch, constraint)| {
            (
                sketch,
                vec![oblikovati_gui::GuiConstraintCandidate {
                    constraint,
                    text: String::new(),
                }],
            )
        })
    }
    /// Check whether the ribbon awaits a model pick; example: `window.has_pending_ribbon_model_pick()`.
    fn has_pending_ribbon_model_pick(&self) -> bool {
        false
    }
    /// Resolve a deferred ribbon click after the model pick; example: `window.resolve_ribbon_model_pick(true)`.
    fn resolve_ribbon_model_pick(&mut self, _face: bool) {}
    /// Supply a confirmed semantic face hit to an armed extrusion picker.
    /// Example: `window.resolve_extrusion_model_pick(&mut || Some((document, face.clone())))`.
    fn resolve_extrusion_model_pick(&mut self, _hit: &mut dyn FnMut() -> Option<(u64, String)>) {}
    /// Open an extrusion draft; example: `window.request_extrusion()`.
    fn request_extrusion(&mut self) {}
    /// Return a rejected draft and its reason; example: `window.reject_dimension_edit(&value, "invalid")`.
    fn reject_dimension_edit(&mut self, _value: &oblikovati_gui::GuiDimensionValue, _cause: &str) {}
    /// Restore a rejected offset-plane draft; example: `window.reject_offset_plane(document, base, offset, cause)`.
    fn reject_offset_plane(
        &mut self,
        _document: u64,
        _base: oblikovati_gui::GuiOffsetPlaneBase,
        _offset: &str,
        _cause: &str,
    ) {
    }
    /// Return an extrusion draft rejected by its owner; example: `window.reject_extrusion(&request, "profile missing")`.
    fn reject_extrusion(&mut self, _request: &oblikovati_gui::GuiExtrudePreview, _cause: &str) {}
    fn reject_extrusion_edit(
        &mut self,
        _request: &oblikovati_gui::GuiExtrudeEditRequest,
        _cause: &str,
    ) {
    }

    /// Replace the active document's committed sketches; example: `window.submit_sketch_scene(update)?` after Undo.
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        if update.sketches.is_empty()
            && update.plane_declines.is_empty()
            && update.block_declines.is_empty()
        {
            return Ok(());
        }
        Err(format!(
            "sketch scene {:?}: expected a window with committed sketch rendering",
            update.source
        ))
    }

    /// Apply edit-plane framing or restore the return view; example: `window.set_sketch_edit(None)?` on finish.
    fn set_sketch_edit(&mut self, edit: Option<crate::HeadSketchEdit>) -> Result<bool, String> {
        if edit.is_none() {
            return Ok(false);
        }
        Err(format!(
            "sketch edit {edit:?}: expected a window with sketch edit framing"
        ))
    }

    /// Replace or clear the authoring outline; example: `window.set_sketch_preview(None)?` after cancellation.
    /// Bring one document window's view forward; example: `window.set_view_slot(owner.view_slot())`.
    fn set_view_slot(&mut self, _slot: u64) -> Result<bool, String> {
        Ok(false)
    }
    fn set_sketch_preview(
        &mut self,
        preview: Option<&oblikovati_viewport::ViewportSketchPreview>,
    ) -> Result<bool, String> {
        if preview.is_none() {
            return Ok(false);
        }
        Err(format!(
            "head sketch preview with {} members: expected a window with outline rendering",
            preview.expect("present preview").member_count()
        ))
    }

    /// Update transient tool feedback; example: `window.set_tool_candidate(None)?` on exit.
    fn set_tool_candidate(
        &mut self,
        _: Option<&oblikovati_viewport::ViewportSketchHover>,
    ) -> Result<bool, String> {
        Ok(false)
    }

    /// Publish authored or derived feedback; example: `window.set_tool_feedback(None)?`.
    fn set_tool_feedback(
        &mut self,
        feedback: Option<&oblikovati_viewport::ViewportSketchFeedback>,
    ) -> Result<bool, String> {
        match feedback {
            Some(oblikovati_viewport::ViewportSketchFeedback::Entities(hover)) => {
                self.set_tool_candidate(Some(hover))
            }
            None => self.set_tool_candidate(None),
            Some(point) => Err(format!(
                "sketch feedback {point:?}: expected a derived-point capable window"
            )),
        }
    }

    /// Apply one view-only input before projecting that same event; example: `window.process_view_input(&event)?`.
    fn process_view_input(&mut self, _: &ViewportEvent) -> Result<bool, String> {
        Ok(false)
    }

    /// Project sketch witnesses; example: `window.sketch_pixels(plane, &points)?` for a pick pass.
    fn sketch_pixels(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        points: &[[f64; 2]],
    ) -> Result<Vec<[i32; 2]>, String> {
        points
            .iter()
            .map(|point| self.sketch_pixel(plane, *point))
            .collect()
    }

    /// Project ordered picks with individual errors; example: `window.sketch_pixel_results(plane, &points)?`.
    fn sketch_pixel_results(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        points: &[[f64; 2]],
    ) -> Result<Vec<Result<[i32; 2], String>>, String> {
        Ok(points
            .iter()
            .map(|point| self.sketch_pixel(plane, *point))
            .collect())
    }

    /// Map physical pointer coordinates through the current view; example: `window.sketch_point(pixel,plane)?`.
    fn sketch_point(
        &self,
        pixel: [i32; 2],
        plane: oblikovati_model::sketch::SketchPlane,
    ) -> Result<Option<[f64; 2]>, String> {
        Err(format!(
            "sketch cursor {pixel:?} on {plane:?}: expected a window with sketch projection"
        ))
    }

    /// Keep modelling clicks out of an active camera gesture.
    /// Example: `window.navigation_active()` before selecting a sketch entity.
    fn navigation_active(&self) -> bool {
        false
    }

    /// Permit discarding an unseen idle model or sketch motion at this physical pixel.
    /// Example: `window.allows_ready_sketch_motion([400, 300])` rejects active gestures.
    fn allows_ready_sketch_motion(&self, _pixel: [i32; 2]) -> bool {
        false
    }

    /// Permit batching GUI input when the viewport owns a camera drag.
    /// Example: `window.batches_navigation_motion()` remains false for a held navigation key alone.
    fn batches_navigation_motion(&self) -> bool {
        false
    }

    /// Project an analytic sketch witness for screen-space hit acceptance.
    /// Example: `window.sketch_pixel(plane, witness.position)?`.
    fn sketch_pixel(
        &self,
        plane: oblikovati_model::sketch::SketchPlane,
        point: [f64; 2],
    ) -> Result<[i32; 2], String> {
        Err(format!(
            "sketch witness {point:?} on {plane:?}: expected a window with sketch projection"
        ))
    }

    /// The camera the view looks through; example: for the orientation control.
    fn view_camera(&self) -> Result<oblikovati_viewport::ViewportCamera, String> {
        Err("view camera: expected a window with a view".into())
    }
    /// Look through this camera instead; example: after a press on the orientation control.
    fn set_view_camera(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<bool, String> {
        Err(format!(
            "view camera {camera:?}: expected a window with a view"
        ))
    }
    /// Change a discrete view and retain its previous camera; example: `window.set_view_camera_recorded(camera)?`.
    fn set_view_camera_recorded(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<bool, String> {
        self.set_view_camera(camera)
    }
    /// Schedule a camera transition. Example: `window.start_orientation_transition(camera, None)?`.
    fn start_orientation_transition(
        &mut self,
        camera: oblikovati_viewport::ViewportCamera,
        finish: Option<oblikovati_viewport::ViewportNavigation>,
    ) -> Result<bool, String> {
        let changed = self.set_view_camera(camera)?;
        match finish {
            Some(command) => self.navigate_view(command),
            None => Ok(changed),
        }
    }
    /// Advance the camera before computing GUI projections. Example: `window.tick_orientation_transition()?`.
    fn tick_orientation_transition(
        &mut self,
    ) -> Result<(bool, Option<std::time::Duration>), String> {
        Ok((false, None))
    }
    /// The line of sight through a pixel, as a point and a direction away from the viewer.
    fn view_line(&self, pixel: [i32; 2]) -> Result<[[f64; 3]; 2], String> {
        Err(format!(
            "view line {pixel:?}: expected a window with a view"
        ))
    }
    /// Move the view by one navigation step; example: a fit, or a drag of the running command.
    fn navigate_view(
        &mut self,
        command: oblikovati_viewport::ViewportNavigation,
    ) -> Result<bool, String> {
        Err(format!(
            "view navigation {command:?}: expected a window with a view"
        ))
    }
    /// Return the drawable extent and unobstructed document rectangle for a selection-centered fit.
    /// Example: `window.orientation_fit_frame()?` after a GUI frame.
    fn orientation_fit_frame(&mut self) -> Result<HeadOrientationFitFrame, String> {
        Err("selection fit area: expected an active native document viewport".into())
    }
    /// Remember this view, for a dialog that may put it back; example: on opening the dialog.
    fn mark_view(&mut self) -> Result<(), String> {
        Err("mark view: expected a window with a view".into())
    }
    /// Put the remembered view back; example: from the rotate dialog's Reset.
    fn restore_marked_view(&mut self) -> Result<bool, String> {
        Err("restore view: expected a window with a view".into())
    }
    /// Step back to the view before the last fit; example: from a navigation command's menu.
    fn previous_view(&mut self) -> Result<bool, String> {
        Err("previous view: expected a window with a view".into())
    }
    /// The projection the view keeps to; example: from the orientation control's menu.
    fn view_projection(&self) -> oblikovati_gui::GuiViewProjection {
        Default::default()
    }
    /// Keep the view to this projection; example: from the orientation control's menu.
    fn set_view_projection(&mut self, _: oblikovati_gui::GuiViewProjection) {}
    /// Cached visible-scene bounds for presentation only; example: size an origin-plane overlay.
    /// These bounds must never be used for modelling or topology decisions.
    fn presentation_bounds(&self) -> Option<[[f64; 3]; 2]> {
        None
    }
    /// The pixel a model position shows at; example: to draw a solid's edges over the view.
    fn model_pixel(&self, point: [f64; 3]) -> Result<[i32; 2], String> {
        Err(format!(
            "model position {point:?}: expected a window with a projection"
        ))
    }

    /// Show an owner command result without terminating the window; example: `window.show_command_notice(notice)?`.
    fn show_command_notice(&mut self, notice: crate::DesktopCommandNotice) -> Result<(), String> {
        Err(format!(
            "head command {notice:?}: expected a window with command feedback"
        ))
    }

    /// Replace work-plane overlays; example: `window.set_datum_planes(&planes)?`.
    fn set_datum_planes(&mut self, update: &crate::HeadDatumPlaneUpdate) -> Result<bool, String> {
        if update.planes.is_empty() && update.declines.is_empty() {
            return Ok(false);
        }
        Err(format!(
            "datum planes {}: expected a window with datum presentation",
            update.planes.len()
        ))
    }

    /// Replace the current scene before presenting; example: `window.submit_scene(update)?`.
    fn submit_scene(&mut self, update: crate::HeadSceneUpdate) -> Result<(), String> {
        Err(format!(
            "head scene {:?}: expected a window with model presentation",
            update.status.source
        ))
    }

    /// Poll bounded lifecycle inputs; example: `window.events(64)?`.
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String>;
    /// Present once without returning a stale image; example: `window.present()?`.
    fn present(&mut self) -> Result<HeadPresentation, String>;
    /// Wait for native input or a worker wake; return true for a requested redraw.
    /// Example: `let requested = window.wait(activity.wake_after)?;` while idle.
    fn wait(&mut self, timeout: Option<std::time::Duration>) -> Result<bool, String>;
    /// Retire the view and check renderer diagnostics; example: `window.finish()?` before process exit.
    fn finish(&mut self) -> Result<(), String>;
}
