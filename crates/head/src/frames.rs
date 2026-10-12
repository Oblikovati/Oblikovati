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
/// The desktop owns this boundary; implementations contain all native I/O.
pub trait HeadWindow {
    /// Poll bounded lifecycle inputs; example: `window.events(64)?`.
    fn events(&mut self, limit: usize) -> Result<Vec<ViewportEvent>, String>;
    /// Present once without returning a stale image; example: `window.present()?`.
    fn present(&mut self) -> Result<HeadPresentation, String>;
    /// Wait for native input or a worker wake; return true for a requested redraw.
    /// Example: `let requested = window.wait()?;` while idle.
    fn wait(&mut self) -> Result<bool, String>;
    /// Retire the view and check renderer diagnostics; example: `window.finish()?` before process exit.
    fn finish(&mut self) -> Result<(), String>;
}
struct FrameControl {
    options: HeadOptions,
    summary: HeadSummary,
    pending: bool,
    mapped: bool,
    closed: bool,
}
impl FrameControl {
    fn new(options: HeadOptions) -> Self {
        Self {
            options,
            summary: HeadSummary::default(),
            pending: true,
            mapped: true,
            closed: false,
        }
    }
    fn stopped(&self) -> bool {
        self.closed
            || self
                .options
                .frames
                .is_some_and(|count| self.summary.attempts >= count)
    }
    fn events(&mut self, events: &[ViewportEvent], close_allowed: bool) {
        for event in events {
            match event {
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
    let outcome = combine_retirement(outcome, owner.finish());
    combine_retirement(outcome, window.finish())
}
fn combine_retirement(
    outcome: Result<HeadSummary, String>,
    retirement: Result<(), String>,
) -> Result<HeadSummary, String> {
    match (outcome, retirement) {
        (Ok(summary), Ok(())) => Ok(summary),
        (Err(error), Ok(())) | (Ok(_), Err(error)) => Err(error),
        (Err(error), Err(cleanup)) => Err(format!("head run: {error}; retirement: {cleanup}")),
    }
}
fn run_frames(
    window: &mut dyn HeadWindow,
    options: HeadOptions,
    owner: &mut dyn HeadWork,
) -> Result<HeadSummary, String> {
    let mut control = FrameControl::new(options);
    while !control.stopped() {
        let inputs = window.events(64)?;
        let activity = owner.poll(32)?;
        control.pending |= activity.changed;
        let closing = inputs.contains(&ViewportEvent::CloseRequested);
        control.events(&inputs, !closing || owner.close_allowed()?);
        if control.stopped() {
            break;
        }
        control.frame(window)?;
        if control.should_wait() && !activity.pending {
            control.pending |= window.wait()?;
        }
    }
    checked_summary(control.summary, options)
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
