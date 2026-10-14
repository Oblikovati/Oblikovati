// SPDX-License-Identifier: GPL-2.0-only
use imgui::{Io, Key, MouseButton};
macro_rules! owned_keyboard_keys {
    ($($key:ident),+ $(,)?) => {
        #[derive(Clone, Copy, Debug, PartialEq, Eq)]
        pub enum GuiKey { $($key),+ }
        impl GuiKey {
            #[cfg(test)]
            pub(crate) const ALL: &[Self] = &[$(Self::$key),+];
            pub(crate) fn native(self) -> Key {
                match self { $(Self::$key => Key::$key),+ }
            }
        }
    };
}
owned_keyboard_keys!(
    Tab,
    LeftArrow,
    RightArrow,
    UpArrow,
    DownArrow,
    PageUp,
    PageDown,
    Home,
    End,
    Insert,
    Delete,
    Backspace,
    Space,
    Enter,
    Escape,
    LeftCtrl,
    LeftShift,
    LeftAlt,
    LeftSuper,
    RightCtrl,
    RightShift,
    RightAlt,
    RightSuper,
    Menu,
    Alpha0,
    Alpha1,
    Alpha2,
    Alpha3,
    Alpha4,
    Alpha5,
    Alpha6,
    Alpha7,
    Alpha8,
    Alpha9,
    A,
    B,
    C,
    D,
    E,
    F,
    G,
    H,
    I,
    J,
    K,
    L,
    M,
    N,
    O,
    P,
    Q,
    R,
    S,
    T,
    U,
    V,
    W,
    X,
    Y,
    Z,
    F1,
    F2,
    F3,
    F4,
    F5,
    F6,
    F7,
    F8,
    F9,
    F10,
    F11,
    F12,
    Apostrophe,
    Comma,
    Minus,
    Period,
    Slash,
    Semicolon,
    Equal,
    LeftBracket,
    Backslash,
    RightBracket,
    GraveAccent,
    CapsLock,
    ScrollLock,
    NumLock,
    PrintScreen,
    Pause,
    Keypad0,
    Keypad1,
    Keypad2,
    Keypad3,
    Keypad4,
    Keypad5,
    Keypad6,
    Keypad7,
    Keypad8,
    Keypad9,
    KeypadDecimal,
    KeypadDivide,
    KeypadMultiply,
    KeypadSubtract,
    KeypadAdd,
    KeypadEnter,
    KeypadEqual,
);
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuiPointerButton {
    Left,
    Middle,
    Right,
}
/// Aggregate modifier state accompanies physical key events, including AltGr combinations.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct GuiModifiers {
    pub control: bool,
    pub shift: bool,
    pub alt: bool,
    pub super_key: bool,
}
#[derive(Clone, Debug, PartialEq)]
pub enum GuiInputEvent {
    Focus(bool),
    PointerMoved([f32; 2]),
    PointerButton {
        button: GuiPointerButton,
        pressed: bool,
    },
    Wheel([f32; 2]),
    Key {
        key: GuiKey,
        pressed: bool,
    },
    Modifiers(GuiModifiers),
    Text(String),
}
/// Dear ImGui's request for each input channel, copied with its frame.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct GuiInputCapture {
    pub pointer: bool,
    pub keyboard: bool,
    pub text: bool,
}
pub(crate) fn validate_input(event: &GuiInputEvent, scale: [f32; 2]) -> Result<(), String> {
    let valid = match event {
        GuiInputEvent::PointerMoved(position) => (0..2)
            .all(|axis| position[axis].is_finite() && (position[axis] / scale[axis]).is_finite()),
        GuiInputEvent::Wheel(wheel) => wheel.iter().all(|value| value.is_finite()),
        GuiInputEvent::Text(text) => !text.contains('\0'),
        _ => true,
    };
    if !valid {
        return Err(format!(
            "GUI input {event:?}, scale {scale:?}: expected finite logical coordinates and text without NUL"
        ));
    }
    Ok(())
}
pub(crate) fn apply_input(io: &mut Io, event: &GuiInputEvent, scale: [f32; 2]) {
    match event {
        GuiInputEvent::Focus(focused) => crate::context_owner::queue_focus(io, *focused),
        GuiInputEvent::PointerMoved(position) => {
            io.add_mouse_pos_event(std::array::from_fn(|axis| position[axis] / scale[axis]))
        }
        GuiInputEvent::PointerButton { button, pressed } => {
            io.add_mouse_button_event(button.native(), *pressed)
        }
        GuiInputEvent::Wheel(wheel) => io.add_mouse_wheel_event(*wheel),
        GuiInputEvent::Key { key, pressed } => io.add_key_event(key.native(), *pressed),
        GuiInputEvent::Modifiers(modifiers) => apply_modifiers(io, *modifiers),
        GuiInputEvent::Text(text) => apply_text(io, text),
    }
}
impl GuiPointerButton {
    fn native(self) -> MouseButton {
        match self {
            Self::Left => MouseButton::Left,
            Self::Middle => MouseButton::Middle,
            Self::Right => MouseButton::Right,
        }
    }
}
fn apply_modifiers(io: &mut Io, modifiers: GuiModifiers) {
    let pressed = [
        modifiers.control,
        modifiers.shift,
        modifiers.alt,
        modifiers.super_key,
    ];
    for (key, pressed) in [Key::ModCtrl, Key::ModShift, Key::ModAlt, Key::ModSuper]
        .into_iter()
        .zip(pressed)
    {
        io.add_key_event(key, pressed);
    }
}
fn apply_text(io: &mut Io, text: &str) {
    for character in text.chars() {
        io.add_input_character(character);
    }
}
pub(crate) fn input_capture(io: &Io) -> GuiInputCapture {
    GuiInputCapture {
        pointer: io.want_capture_mouse,
        keyboard: io.want_capture_keyboard,
        text: io.want_text_input,
    }
}
#[cfg(test)]
#[path = "../tests/input/unit.rs"]
mod tests;
