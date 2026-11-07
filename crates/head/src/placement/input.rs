// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchPlacementRecipe, SketchPlane},
};
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey,
};
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
    line: line::LineChain,
    cursor_point: Option<[f64; 2]>,
    /// Where the numeric fields follow the pointer, in window pixels.
    pub(crate) cursor_pixel: Option<[i32; 2]>,
    pub(crate) formats: oblikovati_api::types::SketchFormatModes,
    /// For Point: what the next click makes, and so what the command asks for.
    pub(crate) center_point: bool,
    /// A typed value becomes a dimension unless the application option turns that off.
    pub(crate) committing_value_dimensions: bool,
    press: Option<PlacementPress>,
    navigation: [bool; 3],
    fields: PlacementFieldEntry,
}
/// A witness's middle, moved off it by a fraction of its length so the text clears the line.
fn annotation_beside([from, to]: [[f64; 2]; 2]) -> [f64; 2] {
    const SIDE: f64 = 0.2;
    let delta = [to[0] - from[0], to[1] - from[1]];
    [
        (from[0] + to[0]) * 0.5 - delta[1] * SIDE,
        (from[1] + to[1]) * 0.5 + delta[0] * SIDE,
    ]
}
/// How far from its first point a shape starts when values were typed with no pointer motion.
const TYPED_SEED_REACH: f64 = 1.;
struct PlacementPress {
    pixel: [i32; 2],
    dragged: bool,
}
pub(crate) enum PlacementAction {
    Idle,
    Cancel,
    FieldsChanged,
    CommitFields,
    FinishLine,
    RestartLine,
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
            line: Default::default(),
            cursor_point: None,
            center_point: false,
            cursor_pixel: None,
            formats: Default::default(),
            committing_value_dimensions: true,
            press: None,
            navigation: [false; 3],
            fields: Default::default(),
        })
    }
    pub(crate) fn event(&mut self, event: ViewportEvent) -> PlacementAction {
        if let Some(action) = self.line_event(event) {
            return action;
        }
        if let ViewportEvent::Input(input) = event {
            let count = self.field_count();
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
    fn field_count(&self) -> usize {
        if self.tool == crate::SketchPlacementTool::Line {
            return self.line_field_count();
        }
        // The fields exist from the stage that can finish, even before the pointer has moved.
        if self.accepted.len() + 1 == self.tool.point_count() {
            return self.tool.typed_field_count();
        }
        0
    }
    /// Whether the numeric fields are up; example: for the field display beside the pointer.
    pub(crate) fn placement_field_count(&self) -> usize {
        self.field_count()
    }
    /// The recipe the typed values apply to, when the pointer has given none of its own yet.
    pub(crate) fn typed_seed_recipe(
        &mut self,
        construction: &dyn oblikovati_model::sketch::SketchCircleConstruction,
    ) -> Result<Option<SketchPlacementRecipe>, String> {
        if self.preview.is_some() {
            return Ok(self.preview.clone());
        }
        let Some(start) = self.accepted.last().copied() else {
            return Ok(None);
        };
        // The solver takes the typed size; this only needs to be a shape it can start from.
        let seed = [start[0] + TYPED_SEED_REACH, start[1] + TYPED_SEED_REACH];
        let mut recipe = self.tool.recipe(&self.accepted, seed, construction)?;
        recipe
            .apply_creation_formats(self.formats.construction, self.formats.centerline)
            .map_err(|cause| cause.to_string())?;
        Ok(Some(recipe))
    }
    /// A typed zero size on a tool that shrugs it off: the shape is dropped, not refused.
    pub(crate) fn typed_size_is_zero(&self) -> bool {
        self.fields.line_values().contains(&Some(0.))
    }
    /// The pointer's sketch position, once it has moved over the sketch.
    pub(crate) fn pointer_position(&self) -> Option<[f64; 2]> {
        self.cursor_point
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
    pub(crate) fn fields_invalid(&self) -> bool {
        self.fields.has_invalid_value()
    }
    pub(crate) fn field_view(&self, index: usize) -> fields::PlacementFieldView {
        self.fields.view(index)
    }
    pub(crate) fn evaluate_line_values(&mut self, evaluate: &dyn Fn(usize, &str) -> Option<f64>) {
        self.fields.evaluate_line_fields(evaluate);
    }
    /// Typed values with where each dimension's text goes: beside the edge it measures.
    pub(crate) fn locks_for(
        &self,
        recipe: &SketchPlacementRecipe,
    ) -> Result<Vec<Option<oblikovati_model::history::SketchPlacementFieldLock>>, String> {
        let mut locks = self.fields.preview_locks();
        fields::validate_dimension_locks(&recipe.fields, &locks)?;
        for (lock, field) in locks.iter_mut().zip(&recipe.fields) {
            if let Some(lock) = lock {
                lock.text_point = Some(annotation_beside(field.witness));
            }
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
        let line = self.tool == crate::SketchPlacementTool::Line;
        if pressed {
            self.press = Some(PlacementPress {
                pixel,
                dragged: false,
            });
            if line {
                return self.line_press(pixel);
            }
            return PlacementAction::Cursor { pixel, place: true };
        }
        self.track_drag(pixel);
        let place = self.press.take().is_some_and(|press| press.dragged);
        if line {
            return self.line_release(place);
        }
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
        // A click where the last one landed is no shape yet: the command keeps asking.
        if self.accepted.last() == Some(&point) {
            return Ok(None);
        }
        if self.accepted.len() + 1 < self.tool.point_count() {
            if place {
                self.accept_point(point)?;
            }
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
            return self.line_prompt();
        }
        if self.tool == crate::SketchPlacementTool::Point {
            return if self.center_point {
                "Select center point".to_owned()
            } else {
                "Select Sketch Point".to_owned()
            };
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
