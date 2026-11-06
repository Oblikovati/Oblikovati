// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::history::SketchLineChainCreation;
use oblikovati_viewport::ViewportPhysicalKey;

impl SketchPlacementState {
    pub(super) fn line_event(&self, event: ViewportEvent) -> Option<PlacementAction> {
        if self.tool != crate::SketchPlacementTool::Line {
            return None;
        }
        match event {
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key:
                    ViewportPhysicalKey::Enter
                    | ViewportPhysicalKey::KeypadEnter
                    | ViewportPhysicalKey::Escape,
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
            }) => Some(PlacementAction::FinishLine { closed: false }),
            ViewportEvent::Input(ViewportInput::Text('c' | 'C')) => {
                Some(PlacementAction::FinishLine { closed: true })
            }
            ViewportEvent::Input(ViewportInput::PhysicalKey {
                key: ViewportPhysicalKey::Backspace,
                pressed: true,
                ..
            })
            | ViewportEvent::Input(ViewportInput::Text('u' | 'U')) => {
                Some(PlacementAction::UndoLine)
            }
            _ => None,
        }
    }
    pub(super) fn line_cursor(
        &mut self,
        point: [f64; 2],
        place: bool,
    ) -> Result<Option<SketchPlacementRecipe>, String> {
        if place {
            self.accept_point(point)?;
            self.line_pending = true;
        }
        self.preview = self
            .line_input(true, false)
            .map(|input| self.formatted_line_recipe(&input))
            .transpose()
            .map_err(|cause| cause.to_string())?;
        Ok(None)
    }
    pub(crate) fn line_input(&self, cursor: bool, closed: bool) -> Option<SketchLineChainCreation> {
        let mut points = self.accepted.clone();
        if cursor
            && let Some(point) = self.cursor_point
            && points.last() != Some(&point)
        {
            points.push(point);
        }
        if points.len() < 2 {
            return None;
        }
        Some(SketchLineChainCreation {
            closed: closed && points.len() >= 3,
            points,
            construction: self.formats.construction,
            centerline: self.formats.centerline,
        })
    }
    pub(crate) fn confirm_line_vertex(&mut self) {
        self.line_pending = false;
    }
    pub(super) fn reject_line_vertex(&mut self) {
        if self.line_pending {
            self.accepted.pop();
        }
        self.line_pending = false;
        self.cursor_point = None;
    }
    pub(crate) fn undo_line_vertex(&mut self) -> Result<(), String> {
        self.accepted.pop();
        self.cursor_point = None;
        self.preview = None;
        self.resolved_preview = None;
        self.preview = self
            .line_input(false, false)
            .map(|input| self.formatted_line_recipe(&input))
            .transpose()
            .map_err(|cause| cause.to_string())?;
        Ok(())
    }
}

#[cfg(test)]
#[path = "line_tests.rs"]
mod tests;
