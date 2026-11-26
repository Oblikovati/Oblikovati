// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    history::{SketchLineChainCreation, SketchLineChainCreationReport},
    sketch::SketchEntityId,
};

/// The chain under construction: every accepted segment is already its own document operation,
/// so this holds only what the next segment joins and what the active menu offers.
#[derive(Default)]
pub(super) struct LineChain {
    start: Option<[f64; 2]>,
    start_vertex: Option<SketchEntityId>,
    first: Option<([f64; 2], SketchEntityId)>,
    segments: usize,
    accepted_pixel: Option<[i32; 2]>,
    repeat_press: bool,
}
/// Within this many pixels, a second click at an accepted endpoint is a double-click. A click
/// three pixels on makes its own short segment, as the recorded repeated-coordinate attempt did.
const REPEAT_CLICK_PIXELS: i32 = 2;

impl SketchPlacementState {
    pub(super) fn line_event(&self, event: ViewportEvent) -> Option<PlacementAction> {
        if self.tool != crate::SketchPlacementTool::Line {
            return None;
        }
        match event {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Escape,
                pressed: true,
                ..
            })
            | ViewportEvent::Input(ViewportInput::Key {
                key: ViewportKey::Cancel,
                pressed: true,
            })
            | ViewportEvent::Input(ViewportInput::PhysicalKey {
                command: Some(ViewportKey::Cancel),
                pressed: true,
                ..
            }) => Some(PlacementAction::FinishLine),
            // Enter with nothing typed accepts the pending end, as the shown values describe it.
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Enter | ViewportPhysicalKey::KeypadEnter,
                pressed: true,
                ..
            }) if !self.fields.engaged() => Some(PlacementAction::CommitFields),
            _ => None,
        }
    }
    /// Keep the pending segment current; a placing press sets the start or requests a commit.
    pub(crate) fn line_cursor(&mut self, point: Option<[f64; 2]>) {
        self.cursor_point = point;
    }
    /// True when this press belongs to a new start point; a commit needs the owner.
    pub(crate) fn line_awaits_start(&self) -> bool {
        self.line.start.is_none()
    }
    /// Start from an exact position, joined to the vertex there when one was acquired.
    pub(crate) fn line_start_on(
        &mut self,
        point: [f64; 2],
        vertex: Option<SketchEntityId>,
        pixel: Option<[i32; 2]>,
    ) -> Result<(), String> {
        if point.iter().any(|value| !value.is_finite()) {
            return Err(format!(
                "Line start {point:?}: expected a finite sketch position"
            ));
        }
        self.line = LineChain {
            start: Some(point),
            start_vertex: vertex,
            accepted_pixel: pixel,
            ..Default::default()
        };
        Ok(())
    }
    /// A press at the accepted endpoint never makes a segment: it is a double-click or a drag.
    pub(super) fn line_press(&mut self, pixel: [i32; 2]) -> PlacementAction {
        self.line.repeat_press = self.line.accepted_pixel.is_some_and(|accepted| {
            (0..2).all(|axis| (accepted[axis] - pixel[axis]).abs() <= REPEAT_CLICK_PIXELS)
        });
        if self.line.repeat_press {
            return PlacementAction::Idle;
        }
        PlacementAction::Cursor { pixel, place: true }
    }
    /// A release never places: a dragged endpoint stays a pending preview for the next click.
    pub(super) fn line_release(&mut self, dragged: bool) -> PlacementAction {
        let repeated = std::mem::take(&mut self.line.repeat_press);
        if repeated && !dragged && self.line.segments > 0 {
            return PlacementAction::RestartLine;
        }
        PlacementAction::Idle
    }
    /// The pending end, with typed values taking over from the pointer.
    pub(crate) fn line_end(&self) -> Option<[f64; 2]> {
        let start = self.line.start?;
        let [length, angle] = self.fields.line_values();
        let cursor = self.cursor_point.unwrap_or(start);
        if length.is_none() && angle.is_none() {
            return (cursor != start).then_some(cursor);
        }
        let delta = [cursor[0] - start[0], cursor[1] - start[1]];
        let direction = angle.unwrap_or_else(|| delta[1].atan2(delta[0]));
        let reach =
            length.unwrap_or_else(|| delta[0] * direction.cos() + delta[1] * direction.sin());
        let end = [
            start[0] + reach * direction.cos(),
            start[1] + reach * direction.sin(),
        ];
        (end != start).then_some(end)
    }
    /// The one-segment operation to preview or commit; example: `state.line_input()`.
    pub(crate) fn line_input(&self) -> Option<SketchLineChainCreation> {
        let start = self.line.start?;
        let end = self.line_end()?;
        let mut locks = self.fields.line_locks(self.committing_value_dimensions);
        for (lock, text_point) in locks.iter_mut().zip(annotation_points(start, end)) {
            if let Some(lock) = lock {
                lock.text_point = Some(text_point);
            }
        }
        Some(SketchLineChainCreation {
            points: vec![start, end],
            closed: false,
            construction: self.formats.construction,
            centerline: self.formats.centerline,
            joins: [self.line.start_vertex, None],
            locks,
        })
    }
    /// The closing edge from the current end back to the chain's first point.
    pub(crate) fn line_close_input(&self) -> Option<SketchLineChainCreation> {
        if !self.line_close_available() {
            return None;
        }
        let (first, first_vertex) = self.line.first?;
        self.line_segment_to(first, Some(first_vertex))
    }
    /// One segment from the current end to an exact point, with nothing typed to lock.
    pub(crate) fn line_segment_to(
        &self,
        end: [f64; 2],
        end_vertex: Option<SketchEntityId>,
    ) -> Option<SketchLineChainCreation> {
        Some(SketchLineChainCreation {
            points: vec![self.line.start?, end],
            closed: false,
            construction: self.formats.construction,
            centerline: self.formats.centerline,
            joins: [self.line.start_vertex, end_vertex],
            locks: Default::default(),
        })
    }
    /// Continue from the accepted end, with empty numeric fields for the next segment.
    pub(crate) fn accept_line_segment(
        &mut self,
        report: &SketchLineChainCreationReport,
        end: [f64; 2],
        pixel: Option<[i32; 2]>,
        generation: u64,
    ) {
        if let (None, Some(start), Some(vertex)) =
            (self.line.first, self.line.start, report.points.first())
        {
            self.line.first = Some((start, *vertex));
        }
        self.line.start = Some(end);
        self.line.start_vertex = report.points.last().copied();
        self.line.segments += 1;
        self.line.accepted_pixel = pixel;
        self.fields = Default::default();
        self.resolved_preview = None;
        self.preview = None;
        self.generation = generation;
    }
    /// Drop the pending segment and ask for a new start; accepted segments stay.
    pub(crate) fn restart_line(&mut self, generation: u64) {
        self.line = Default::default();
        self.restart(generation);
    }
    /// Refresh the pending segment's shown fields; example: after any pointer or field change.
    pub(crate) fn refresh_line_preview(&mut self) -> Result<(), String> {
        self.preview = self
            .line_input()
            .map(|input| self.formatted_line_recipe(&input))
            .transpose()
            .map_err(|cause| cause.to_string())?;
        Ok(())
    }
    /// A segment joins nothing until the second accepted segment leaves a loop to close.
    pub(crate) fn line_close_available(&self) -> bool {
        self.line.segments >= 2
    }
    pub(crate) fn line_restart_available(&self) -> bool {
        self.line.start.is_some()
    }
    pub(crate) fn line_field_count(&self) -> usize {
        if self.line.start.is_some() { 2 } else { 0 }
    }
    pub(super) fn line_prompt(&self) -> String {
        if self.line.start.is_some() {
            return "Select end of line, or enter a value".into();
        }
        "Select start of line".into()
    }
}

/// Length beside the segment's middle and angle nearer its start, on opposite sides.
fn annotation_points(start: [f64; 2], end: [f64; 2]) -> [[f64; 2]; 2] {
    const SIDE: f64 = 0.2;
    let delta = [end[0] - start[0], end[1] - start[1]];
    let normal = [-delta[1] * SIDE, delta[0] * SIDE];
    [
        [
            start[0] + delta[0] * 0.5 + normal[0],
            start[1] + delta[1] * 0.5 + normal[1],
        ],
        [
            start[0] + delta[0] * 0.3 - normal[0],
            start[1] + delta[1] * 0.3 - normal[1],
        ],
    ]
}

#[cfg(test)]
#[path = "line_tests.rs"]
mod tests;
