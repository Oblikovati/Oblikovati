// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchPlacementRecipe, SketchPlane},
};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput, ViewportKey};
#[path = "fields.rs"]
mod fields;
use fields::{PlacementFieldAction, PlacementFieldEntry};
#[path = "formats.rs"]
mod formats;
#[path = "line.rs"]
mod line;
pub(crate) struct SketchPlacementState {
    pub document: u64,
    pub sketch: DocumentSketchId,
    pub generation: u64,
    pub plane: SketchPlane,
    pub tool: crate::SketchPlacementTool,
    pub preview: Option<SketchPlacementRecipe>,
    pub resolved_preview: Option<oblikovati_viewport::ViewportSketchPreview>,
    accepted: Vec<[f64; 2]>,
    pub line_scope: Option<oblikovati_model::history::TransactionToken>,
    cursor_point: Option<[f64; 2]>,
    pub(crate) formats: oblikovati_api::types::SketchFormatModes,
    line_pending: bool,
    press: Option<PlacementPress>,
    navigation: [bool; 3],
    fields: PlacementFieldEntry,
}
struct PlacementPress {
    pixel: [i32; 2],
    dragged: bool,
}
pub(crate) enum PlacementAction {
    Idle,
    Cancel,
    FieldsChanged,
    CommitFields,
    FinishLine { closed: bool },
    UndoLine,
    Cursor { pixel: [i32; 2], place: bool },
}
impl SketchPlacementState {
    pub(crate) fn new(
        document: u64,
        sketch: DocumentSketchId,
        generation: u64,
        plane: SketchPlane,
        tool: crate::SketchPlacementTool,
    ) -> Result<Self, String> {
        tool.validate()?;
        Ok(Self {
            document,
            sketch,
            generation,
            plane,
            tool,
            preview: None,
            resolved_preview: None,
            accepted: Vec::new(),
            line_scope: None,
            cursor_point: None,
            formats: Default::default(),
            line_pending: false,
            press: None,
            navigation: [false; 3],
            fields: Default::default(),
        })
    }
    pub(crate) fn event(&mut self, event: ViewportEvent) -> PlacementAction {
        if let Some(action) = self.line_event(event) {
            return action;
        }
        if self.tool != crate::SketchPlacementTool::Line
            && let ViewportEvent::Input(input) = event
        {
            let count = self
                .preview
                .as_ref()
                .map_or(0, |recipe| recipe.fields.len());
            match self.fields.input(input, count) {
                PlacementFieldAction::Changed => return PlacementAction::FieldsChanged,
                PlacementFieldAction::Commit => return PlacementAction::CommitFields,
                PlacementFieldAction::Ignored => {}
            }
        }
        match event {
            ViewportEvent::Input(input) => self.input(input),
            ViewportEvent::Focus(false)
            | ViewportEvent::Mapped(false)
            | ViewportEvent::CloseRequested => PlacementAction::Cancel,
            _ => PlacementAction::Idle,
        }
    }
    pub(crate) fn field_prompt(&self) -> String {
        let stage = self.prompt();
        if self.tool == crate::SketchPlacementTool::Line {
            return stage;
        }
        self.preview
            .as_ref()
            .map(|recipe| format!("{stage}  {}", self.fields.prompt(&recipe.fields)))
            .unwrap_or(stage)
    }
    pub(crate) fn fields_engaged(&self) -> bool {
        self.fields.engaged()
    }
    pub(crate) fn preview_locks(
        &self,
    ) -> Result<Vec<Option<oblikovati_model::history::SketchPlacementFieldLock>>, String> {
        let locks = self.fields.preview_locks();
        if let Some(recipe) = &self.preview {
            fields::validate_dimension_locks(&recipe.fields, &locks)?;
        }
        Ok(locks)
    }
    fn input(&mut self, input: ViewportInput) -> PlacementAction {
        match input {
            ViewportInput::Key { key, pressed } => self.key(key, pressed),
            ViewportInput::PhysicalKey {
                command: Some(key),
                pressed,
                ..
            } => self.key(key, pressed),
            ViewportInput::PhysicalKey { .. } | ViewportInput::Text(_) => PlacementAction::Idle,
            ViewportInput::Wheel { .. } => {
                self.press = None;
                PlacementAction::Idle
            }
            ViewportInput::Motion { position, .. } => self.motion(position),
            ViewportInput::Button {
                button,
                pressed,
                position,
                ..
            } => self.button(button, pressed, position),
        }
    }
    fn key(&mut self, key: ViewportKey, pressed: bool) -> PlacementAction {
        let index = match key {
            ViewportKey::Pan => Some(0),
            ViewportKey::Zoom => Some(1),
            ViewportKey::Orbit => Some(2),
            _ => None,
        };
        if let Some(index) = index {
            self.navigation[index] = pressed;
            self.press = None;
        }
        if pressed && matches!(key, ViewportKey::Home | ViewportKey::Previous) {
            self.press = None;
        }
        if key == ViewportKey::Cancel && pressed {
            return PlacementAction::Cancel;
        }
        PlacementAction::Idle
    }
    fn motion(&mut self, pixel: [i32; 2]) -> PlacementAction {
        if self.navigation.contains(&true) {
            return PlacementAction::Idle;
        }
        self.track_drag(pixel);
        PlacementAction::Cursor {
            pixel,
            place: false,
        }
    }
    fn button(
        &mut self,
        button: ViewportButton,
        pressed: bool,
        pixel: [i32; 2],
    ) -> PlacementAction {
        if button != ViewportButton::Left || self.navigation.contains(&true) {
            self.press = None;
            return PlacementAction::Idle;
        }
        if pressed {
            self.press = Some(PlacementPress {
                pixel,
                dragged: false,
            });
            return PlacementAction::Cursor { pixel, place: true };
        }
        self.track_drag(pixel);
        let place = self.press.take().is_some_and(|press| press.dragged);
        if place {
            return PlacementAction::Cursor { pixel, place: true };
        }
        PlacementAction::Idle
    }
    fn track_drag(&mut self, pixel: [i32; 2]) {
        let Some(press) = &mut self.press else {
            return;
        };
        let [dx, dy] = std::array::from_fn(|i| f64::from(pixel[i]) - f64::from(press.pixel[i]));
        // Go #2014: strictly more than four framebuffer pixels turns release into the next click.
        press.dragged |= dx * dx + dy * dy > 16.;
    }
    pub(crate) fn cursor(
        &mut self,
        point: Option<[f64; 2]>,
        place: bool,
        construction: &dyn oblikovati_model::sketch::SketchCircleConstruction,
    ) -> Result<Option<SketchPlacementRecipe>, String> {
        self.resolved_preview = None;
        self.preview = None;
        self.cursor_point = point;
        let Some(point) = point else {
            self.press = None;
            return Ok(None);
        };
        if self.tool == crate::SketchPlacementTool::Line {
            return self.line_cursor(point, place);
        }
        if self.accepted.len() + 1 < self.tool.point_count() {
            if place {
                self.accept_point(point)?;
            }
            return Ok(None);
        }
        if !place && self.accepted.last() == Some(&point) {
            return Ok(None);
        }
        let mut recipe = self.tool.recipe(&self.accepted, point, construction)?;
        recipe
            .apply_creation_formats(self.formats.construction, self.formats.centerline)
            .map_err(|cause| cause.to_string())?;
        self.preview = Some(recipe.clone());
        Ok(place.then_some(recipe))
    }
    pub(crate) fn abandon_press(&mut self) {
        self.reject_line_vertex();
        self.resolved_preview = None;
        self.press = None;
        self.preview = None;
    }
    pub(crate) fn restart(&mut self, generation: u64) {
        self.resolved_preview = None;
        self.fields = Default::default();
        self.generation = generation;
        self.accepted.clear();
        self.cursor_point = None;
        self.press = None;
        self.preview = None;
    }
    pub(crate) fn prompt(&self) -> String {
        if self.tool == crate::SketchPlacementTool::Line {
            return format!(
                "Line: {} points. Enter/Escape finishes; C closes; Backspace undoes a point.",
                self.accepted.len()
            );
        }
        self.tool.prompt(self.accepted.len())
    }
    fn accept_point(&mut self, point: [f64; 2]) -> Result<(), String> {
        if point.iter().any(|value| !value.is_finite()) || self.accepted.last() == Some(&point) {
            return Err(format!(
                "placement point {point:?}: expected a finite point distinct from the preceding point"
            ));
        }
        self.accepted.push(point);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn nonfinite_first_points_do_not_advance_the_placement_stage() {
        let mut state = SketchPlacementState::new(
            1,
            DocumentSketchId(1),
            0,
            SketchPlane::xy(),
            crate::SketchPlacementTool::Rectangle,
        )
        .unwrap();
        let construction = oblikovati_martensite::planar_curve::MartensiteSketchConstruction;
        let before = state.prompt();
        assert!(
            state
                .cursor(Some([f64::NAN, 0.]), true, &construction)
                .is_err()
        );
        assert_eq!(state.prompt(), before);
        assert!(state.preview.is_none());
        state.cursor(Some([0., 0.]), true, &construction).unwrap();
        assert!(
            state
                .cursor(Some([1., 2.]), true, &construction)
                .unwrap()
                .is_some()
        );
    }
}

#[cfg(test)]
#[path = "input_tests.rs"]
mod interaction_tests;
