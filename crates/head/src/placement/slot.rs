// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{SketchPlacementRecipe, SketchPlane},
};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput, ViewportKey};
pub(crate) struct SlotPlacement {
    pub document: u64,
    pub sketch: DocumentSketchId,
    pub generation: u64,
    pub plane: SketchPlane,
    pub width: f64,
    pub preview: Option<SketchPlacementRecipe>,
    start: Option<[f64; 2]>,
    press: Option<PlacementPress>,
    navigation: [bool; 3],
}
struct PlacementPress {
    pixel: [i32; 2],
    dragged: bool,
}
pub(crate) enum PlacementAction {
    Idle,
    Cancel,
    Cursor { pixel: [i32; 2], place: bool },
}
impl SlotPlacement {
    pub(crate) fn new(
        document: u64,
        sketch: DocumentSketchId,
        generation: u64,
        plane: SketchPlane,
        width: f64,
    ) -> Result<Self, String> {
        if !width.is_finite() || width <= 0. {
            return Err(format!(
                "slot width {width}: expected a positive finite working-unit length"
            ));
        }
        Ok(Self {
            document,
            sketch,
            generation,
            plane,
            width,
            preview: None,
            start: None,
            press: None,
            navigation: [false; 3],
        })
    }
    pub(crate) fn event(&mut self, event: ViewportEvent) -> PlacementAction {
        match event {
            ViewportEvent::Input(input) => self.input(input),
            ViewportEvent::Focus(false)
            | ViewportEvent::Mapped(false)
            | ViewportEvent::CloseRequested => PlacementAction::Cancel,
            _ => PlacementAction::Idle,
        }
    }
    fn input(&mut self, input: ViewportInput) -> PlacementAction {
        match input {
            ViewportInput::Key { key, pressed } => self.key(key, pressed),
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
    ) -> Result<Option<SketchPlacementRecipe>, String> {
        self.preview = None;
        let Some(point) = point else {
            self.press = None;
            return Ok(None);
        };
        let Some(start) = self.start else {
            if place {
                self.start = Some(point);
            }
            return Ok(None);
        };
        if !place && start == point {
            return Ok(None);
        }
        let recipe = SketchPlacementRecipe::straight_slot([start, point], self.width)
            .map_err(|cause| cause.to_string())?;
        self.preview = Some(recipe.clone());
        Ok(place.then_some(recipe))
    }
    pub(crate) fn abandon_press(&mut self) {
        self.press = None;
        self.preview = None;
    }
    pub(crate) fn restart(&mut self, generation: u64) {
        self.generation = generation;
        self.start = None;
        self.press = None;
        self.preview = None;
    }
    pub(crate) fn prompt(&self) -> String {
        if self.start.is_none() {
            return "Slot: select the first centre point. Escape cancels.".into();
        }
        "Slot: select the second centre point. Escape cancels.".into()
    }
}
