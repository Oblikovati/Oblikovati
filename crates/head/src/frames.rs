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
mod ready_motion;
pub use gui::HeadGuiUpdate;
mod window;
pub use window::{HeadOrientationFitFrame, HeadWindow};
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
    control.pending |= window.set_datum_planes(&owner.datum_planes()?)?;
    control.pending |= synchronize_sketch_view(window, owner)?;
    control.pending |= publish_sketch_preview(window, owner)?;
    let closing = inputs.contains(&ViewportEvent::CloseRequested);
    let close_allowed = !closing || owner.close_allowed()?;
    if closing && !close_allowed {
        window.request_application_close_confirmation()?;
        activity.pending = true;
    }
    control.events(&inputs, close_allowed);
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
    let mut changed = owner.process_live_window_inputs(&[], window)?;
    changed |= synchronize_sketch_view(window, owner)?;
    changed |= gui::advance_gui(window, owner, None)?.changed;
    let mut index = 0;
    while index < inputs.len() {
        let end = ready_motion::tail(inputs, index, |pixel| {
            window.allows_ready_sketch_motion(pixel)
        });
        if end > index && ready_motion::owner_is_ready(owner)? {
            index = end;
        }
        let event = inputs[index];
        index += 1;
        if event == ViewportEvent::CloseRequested {
            continue;
        }
        let event = owner.resolve_shortcut(event);
        let gui = gui::advance_gui(window, owner, Some(&event))?;
        let shortcut = owner.take_shortcut_action();
        let shortcut = match shortcut {
            Some(oblikovati_gui::GuiShellAction::OpenExtrude)
                if !oblikovati_gui::GuiShellAction::OpenExtrude
                    .dispatchable(&owner.gui_command_state()?) =>
            {
                None
            }
            other => other,
        };
        changed |= gui.changed;
        if gui.consumed_input {
            changed |= window.set_tool_feedback(None)?;
        }
        if !gui.consumed_input {
            if let Some(action) = shortcut {
                changed |= gui::dispatch_shortcut_action(window, owner, &action)?;
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
    // A previous pointer hit belongs to the old sketch geometry (#4043).
    window.set_tool_feedback(None)?;
    window.submit_sketch_scene(update)?;
    Ok(true)
}

#[cfg(test)]
pub(crate) fn advance_gui_for_native_test(
    window: &mut dyn HeadWindow,
    owner: &mut dyn HeadWork,
    event: Option<&ViewportEvent>,
) -> Result<HeadGuiUpdate, String> {
    gui::advance_gui(window, owner, event)
}
