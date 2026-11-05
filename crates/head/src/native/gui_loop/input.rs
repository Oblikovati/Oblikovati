// SPDX-License-Identifier: GPL-2.0-only
#[cfg(test)]
use oblikovati_gui::GuiModifiers;
use oblikovati_gui::{GuiInputCapture, GuiInputEvent, GuiPointerButton};
use oblikovati_viewport::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportModifiers,
};
#[derive(Default)]
pub(super) struct PointerOwnership {
    buttons: [Option<bool>; 3],
}
impl PointerOwnership {
    pub fn consumed(&mut self, event: Option<&ViewportEvent>, capture: GuiInputCapture) -> bool {
        match event {
            Some(ViewportEvent::Input(input)) => self.input(*input, capture),
            Some(
                ViewportEvent::Focus(false)
                | ViewportEvent::Mapped(false)
                | ViewportEvent::CloseRequested,
            ) => {
                self.buttons = [None; 3];
                false
            }
            _ => false,
        }
    }
    fn input(&mut self, input: ViewportInput, capture: GuiInputCapture) -> bool {
        match input {
            ViewportInput::Button {
                button, pressed, ..
            } => self.button(button, pressed, capture.pointer),
            ViewportInput::Motion { .. } => {
                if self.buttons.contains(&Some(false)) {
                    return false;
                }
                self.buttons.contains(&Some(true)) || capture.pointer
            }
            ViewportInput::Wheel { .. } => capture.pointer,
            ViewportInput::Key { key, pressed } => pressed && capture.text && !global_chord(key),
            ViewportInput::PhysicalKey {
                command: Some(key),
                pressed,
                ..
            } => pressed && capture.text && !global_chord(key),
            ViewportInput::PhysicalKey { pressed, .. } => pressed && capture.keyboard,
            ViewportInput::Text(_) => capture.text,
        }
    }
    fn button(&mut self, button: ViewportButton, pressed: bool, capture: bool) -> bool {
        let index = match button {
            ViewportButton::Left => 0,
            ViewportButton::Middle => 1,
            ViewportButton::Right => 2,
        };
        if pressed {
            let owner = *self.buttons[index].get_or_insert(capture);
            return owner;
        }
        self.buttons[index].take().unwrap_or(capture)
    }
}
fn global_chord(key: ViewportKey) -> bool {
    // Go head/ui/chrome.go: Ctrl chords and universal Escape survive text focus.
    matches!(
        key,
        ViewportKey::Save | ViewportKey::Undo | ViewportKey::Redo | ViewportKey::Cancel
    )
}
pub(super) fn copied_gui_events(event: Option<&ViewportEvent>) -> Vec<GuiInputEvent> {
    match event {
        Some(ViewportEvent::Input(input)) => pointer_events(*input),
        Some(ViewportEvent::Focus(focused)) => {
            vec![GuiInputEvent::Focus(*focused)]
        }
        Some(ViewportEvent::CloseRequested | ViewportEvent::Mapped(false)) => {
            vec![GuiInputEvent::Focus(false)]
        }
        _ => Vec::new(),
    }
}
fn pointer_events(input: ViewportInput) -> Vec<GuiInputEvent> {
    match input {
        ViewportInput::Motion {
            position,
            modifiers,
        } => positioned(position, modifiers),
        ViewportInput::Button {
            button,
            pressed,
            position,
            modifiers,
        } => {
            let mut events = positioned(position, modifiers);
            let button = match button {
                ViewportButton::Left => GuiPointerButton::Left,
                ViewportButton::Middle => GuiPointerButton::Middle,
                ViewportButton::Right => GuiPointerButton::Right,
            };
            events.push(GuiInputEvent::PointerButton { button, pressed });
            events
        }
        ViewportInput::Wheel { notches, position } => vec![
            GuiInputEvent::PointerMoved(position.map(|v| v as f32)),
            GuiInputEvent::Wheel([0., notches as f32]),
        ],
        // These are already resolved application commands, not physical key events.
        ViewportInput::Key { .. } => Vec::new(),
        ViewportInput::PhysicalKey {
            layout_key: key,
            pressed,
            modifiers,
            ..
        } => vec![
            GuiInputEvent::Modifiers(modifiers),
            GuiInputEvent::Key { key, pressed },
        ],
        ViewportInput::Text(character) => vec![GuiInputEvent::Text(character.to_string())],
    }
}
fn positioned(position: [i32; 2], modifiers: ViewportModifiers) -> Vec<GuiInputEvent> {
    vec![
        GuiInputEvent::PointerMoved(position.map(|v| v as f32)),
        GuiInputEvent::Modifiers(modifiers),
    ]
}

#[cfg(test)]
#[path = "../../../tests/gui_loop_input.rs"]
mod tests;
