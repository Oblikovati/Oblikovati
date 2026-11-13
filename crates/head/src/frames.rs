// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadWork, work_idle::IdleHeadWork};
use oblikovati_viewport::ViewportEvent;
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum HeadPresentation {
    Completed,
    SurfaceChanged,
}
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct HeadSummary {
    pub attempts: u64,
    pub completed: u64,
}
mod gui;
pub use gui::HeadGuiUpdate;
/// The desktop owns this boundary; implementations contain all native I/O.
pub trait HeadWindow {
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
    /// Picture the viewport for the Save As preview; example: `window.capture_window_preview()?`.
    fn capture_window_preview(&mut self) -> Result<bool, String> {
        Err("window preview: expected a native window that presents frames".into())
    }
    /// Ask the window to close through its own close policy; example: the File menu's Exit.
    fn request_close(&mut self) -> Result<bool, String> {
        Err("exit: expected a native window that can close".into())
    }
    /// Advance GUI work before routing one input; example: `window.gui_frame(Some(event), &commands)?`.
    fn gui_frame(
        &mut self,
        _: Option<&ViewportEvent>,
        _: &oblikovati_gui::GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        Ok(HeadGuiUpdate::default())
    }
    /// Return a rejected draft and its reason; example: `window.reject_dimension_edit(&value, "invalid")`.
    fn reject_dimension_edit(&mut self, _value: &oblikovati_gui::GuiDimensionValue, _cause: &str) {}

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

    /// Apply one view-only input before projecting that same event; example: `window.process_view_input(&event)?`.
    fn process_view_input(&mut self, _: &ViewportEvent) -> Result<bool, String> {
        Ok(false)
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
struct FrameControl {
    options: HeadOptions,
    summary: HeadSummary,
    pending: bool,
    mapped: bool,
    closed: bool,
    startup_ready: bool,
    bounded_attempts: u64,
}
impl FrameControl {
    fn new(options: HeadOptions) -> Self {
        Self {
            options,
            summary: HeadSummary::default(),
            pending: true,
            mapped: true,
            closed: false,
            startup_ready: true,
            bounded_attempts: 0,
        }
    }
    fn stopped(&self) -> bool {
        self.closed
            || self
                .options
                .frames
                .is_some_and(|count| self.bounded_attempts >= count)
    }
    fn events(&mut self, events: &[ViewportEvent], close_allowed: bool) {
        for event in events {
            match event {
                ViewportEvent::Input(_) => {}
                ViewportEvent::CloseRequested => {
                    self.closed = close_allowed;
                    self.pending |= !close_allowed;
                }
                ViewportEvent::Mapped(mapped) => {
                    self.mapped = *mapped;
                    self.pending = *mapped;
                }
                ViewportEvent::Expose | ViewportEvent::Resized(_) | ViewportEvent::Focus(_) => {
                    self.pending = true
                }
            }
        }
    }
    fn frame(&mut self, window: &mut dyn HeadWindow) -> Result<(), String> {
        if !self.mapped && self.options.frames.is_some() {
            return Err("head bounded smoke: expected a mapped native window".into());
        }
        if self.mapped && (self.pending || self.options.frames.is_some()) {
            self.presented(window.present()?)?;
        }
        Ok(())
    }
    fn should_wait(&self) -> bool {
        !self.stopped() && !(self.mapped && self.pending) && self.options.frames.is_none()
    }
    fn presented(&mut self, outcome: HeadPresentation) -> Result<(), String> {
        self.summary.attempts = self.summary.attempts.checked_add(1).ok_or_else(|| {
            format!(
                "head frame count {}: expected an unused u64 count",
                self.summary.attempts
            )
        })?;
        if self.startup_ready {
            self.bounded_attempts += 1;
        }
        self.pending = outcome == HeadPresentation::SurfaceChanged;
        if outcome == HeadPresentation::Completed {
            self.summary.completed += 1;
        }
        Ok(())
    }
}
/// Run the owner loop and always retire the window, including failure paths.
/// Example: `run_window(&mut window, HeadOptions { frames: Some(3) })?` bounds a smoke run.
pub fn run_window(
    window: &mut dyn HeadWindow,
    options: HeadOptions,
) -> Result<HeadSummary, String> {
    run_window_with_owner(window, options, &mut IdleHeadWork)
}
/// Run the same native loop with document-owned services; example: `run_window_with_owner(window, options, owner)?`.
pub fn run_window_with_owner(
    window: &mut dyn HeadWindow,
    options: HeadOptions,
    owner: &mut dyn HeadWork,
) -> Result<HeadSummary, String> {
    let outcome = run_frames(window, options, owner);
    retire_head_run(outcome, owner, window)
}
pub(crate) fn retire_head_run(
    outcome: Result<HeadSummary, String>,
    owner: &mut dyn HeadWork,
    window: &mut dyn HeadWindow,
) -> Result<HeadSummary, String> {
    let outcome = combine_retirement(outcome, owner.finish());
    combine_retirement(outcome, window.finish())
}
pub(crate) fn combine_retirement<T>(
    outcome: Result<T, String>,
    retirement: Result<(), String>,
) -> Result<T, String> {
    match (outcome, retirement) {
        (Ok(summary), Ok(())) => Ok(summary),
        (Err(error), Ok(())) | (Ok(_), Err(error)) => Err(error),
        (Err(error), Err(cleanup)) => Err(format!("head run: {error}; retirement: {cleanup}")),
    }
}
pub(crate) fn run_frames(
    window: &mut dyn HeadWindow,
    options: HeadOptions,
    owner: &mut dyn HeadWork,
) -> Result<HeadSummary, String> {
    let mut control = FrameControl::new(options);
    while !control.stopped() {
        let activity = run_owner_frame(window, owner, &mut control)?;
        if control.should_wait() && !activity.pending {
            control.pending |= window.wait(activity.wake_after)?;
        }
    }
    checked_summary(control.summary, options)
}
fn run_owner_frame(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    control: &mut FrameControl,
) -> Result<crate::HeadWorkStatus, String> {
    let inputs = window.events(64)?;
    owner.observe_window_capture_extent(window.window_capture_extent()?);
    control.pending |= process_frame_inputs(window, owner, &inputs)?;
    let mut activity = owner.poll(32)?;
    publish_command_notice(window, owner)?;
    let gui = gui::advance_gui(window, owner, None)?;
    control.pending |= gui.changed;
    // Deferred GUI input can dispatch after polling; drain its owner effects before idle.
    activity.pending |= !gui.actions.is_empty();
    activity.wake_after = gui::earliest_wake(activity.wake_after, gui.wake_after);
    control.startup_ready = owner.startup_ready();
    control.pending |= activity.changed;
    control.pending |= publish_owner_sketches(window, owner)?;
    control.pending |= publish_owner_scene(window, owner)?;
    control.pending |= synchronize_sketch_view(window, owner)?;
    control.pending |= publish_sketch_preview(window, owner)?;
    let closing = inputs.contains(&ViewportEvent::CloseRequested);
    control.events(&inputs, !closing || owner.close_allowed()?);
    if control.stopped() || gui.deferred {
        return Ok(activity);
    }
    if control.mapped {
        control.pending |= publish_window_capture_requests(window, owner)?;
    }
    control.frame(window)?;
    Ok(activity)
}
fn publish_window_capture_requests(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
) -> Result<bool, String> {
    if !window.window_capture_ready() {
        return Ok(false);
    }
    let requests = owner.take_window_capture_requests();
    if requests.is_empty() {
        return Ok(false);
    }
    window.request_window_captures(requests)?;
    Ok(true)
}
fn process_frame_inputs(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    inputs: &[ViewportEvent],
) -> Result<bool, String> {
    let mut changed = owner.process_window_inputs(&[], window)?;
    changed |= synchronize_sketch_view(window, owner)?;
    changed |= gui::advance_gui(window, owner, None)?.changed;
    for event in inputs {
        let event = owner.resolve_shortcut(*event);
        let gui = gui::advance_gui(window, owner, Some(&event))?;
        let shortcut = owner.take_shortcut_action();
        changed |= gui.changed;
        if !gui.consumed_input {
            if let Some(action) = shortcut {
                changed |= gui::dispatch_gui_action(window, owner, &action)?;
            } else {
                let (consumed, action) = owner.accept_sketch_alias(event);
                if let Some(action) = action {
                    changed |= gui::dispatch_gui_action(window, owner, &action)?;
                } else if consumed {
                    changed = true;
                } else {
                    changed |= gui::dispatch_frame_input(window, owner, &event)?;
                }
            }
        }
    }
    Ok(changed)
}
fn checked_summary(summary: HeadSummary, options: HeadOptions) -> Result<HeadSummary, String> {
    if options.frames.is_some() && summary.completed == 0 {
        return Err(format!(
            "head smoke {:?}: expected at least one completed native frame",
            summary
        ));
    }
    Ok(summary)
}
#[cfg(test)]
#[path = "../tests/frames.rs"]
mod tests;

fn publish_owner_scene(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
) -> Result<bool, String> {
    let Some(update) = owner.take_scene()? else {
        return Ok(false);
    };
    window.submit_scene(update)?;
    Ok(true)
}

fn publish_command_notice(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
) -> Result<(), String> {
    if let Some(notice) = owner.take_command_notice() {
        window.show_command_notice(notice)?;
    }
    Ok(())
}

fn publish_sketch_preview(
    window: &mut dyn HeadWindow,
    owner: &dyn HeadWork,
) -> Result<bool, String> {
    match window.set_sketch_preview(owner.sketch_preview().as_ref()) {
        Ok(changed) => Ok(changed),
        Err(cause) => {
            window.set_sketch_preview(None)?;
            window.show_command_notice(crate::DesktopCommandNotice::PreviewFailed { cause })?;
            Ok(true)
        }
    }
}

fn synchronize_sketch_view(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
) -> Result<bool, String> {
    let switched = window.set_view_slot(owner.view_slot())?;
    match window.set_sketch_edit(owner.sketch_edit()) {
        Ok(changed) => Ok(changed || switched),
        Err(cause) => {
            owner.decline_sketch_edit(cause)?;
            window.set_sketch_edit(None)?;
            Ok(true)
        }
    }
}

fn publish_owner_sketches(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
) -> Result<bool, String> {
    let Some(update) = owner.take_sketch_scene()? else {
        return Ok(false);
    };
    window.submit_sketch_scene(update)?;
    Ok(true)
}
