// SPDX-License-Identifier: GPL-2.0-only
use crate::{HeadOptions, HeadWork, work::IdleHeadWork};
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
    /// Capture the current composed window on the owner thread; example: `window.capture_window()?`.
    fn capture_window(&self) -> Result<oblikovati_viewport::ViewportCapture, String> {
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

    /// Advance GUI work before routing one input; example: `window.gui_frame(Some(event), &commands)?`.
    fn gui_frame(
        &mut self,
        _: Option<&ViewportEvent>,
        _: &oblikovati_gui::GuiCommandState,
    ) -> Result<HeadGuiUpdate, String> {
        Ok(HeadGuiUpdate::default())
    }

    /// Replace the active document's committed sketches; example: `window.submit_sketch_scene(update)?` after Undo.
    fn submit_sketch_scene(&mut self, update: crate::HeadSketchSceneUpdate) -> Result<(), String> {
        if update.sketches.is_empty() && update.plane_declines.is_empty() {
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
    control.pending |= process_frame_inputs(window, owner, &inputs)?;
    let mut activity = owner.poll_window(32, window)?;
    publish_command_notice(window, owner)?;
    let gui = gui::advance_gui(window, owner, None)?;
    control.pending |= gui.changed;
    // Deferred GUI input can dispatch after polling; drain its owner effects before idle.
    activity.pending |= !gui.actions.is_empty() || !gui.parameter_actions.is_empty();
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
    control.frame(window)?;
    Ok(activity)
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
        let gui = gui::advance_gui(window, owner, Some(event))?;
        changed |= gui.changed;
        if !gui.consumed_input {
            changed |= gui::dispatch_frame_input(window, owner, event)?;
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
    match window.set_sketch_edit(owner.sketch_edit()) {
        Ok(changed) => Ok(changed),
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
