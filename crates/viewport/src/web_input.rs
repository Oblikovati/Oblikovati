// SPDX-License-Identifier: GPL-2.0-only
//! Portable browser DOM event translation into Oblikovati viewport events.
use crate::{
    ViewportButton, ViewportEvent, ViewportInput, ViewportKey, ViewportModifiers,
    ViewportPhysicalKey,
};

/// Browser pointer event kind translated from W3C Pointer Events.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum WebPointerKind {
    Down,
    Up,
    Move,
}

/// Copied browser keyboard modifier flags.
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct WebModifiers {
    pub shift: bool,
    pub control: bool,
    pub alt: bool,
    pub meta: bool,
}

impl WebModifiers {
    /// Convert browser modifier flags into Oblikovati viewport modifiers.
    /// Example: `let mods = WebModifiers::default().into_viewport();`.
    pub fn into_viewport(self) -> ViewportModifiers {
        ViewportModifiers {
            shift: self.shift,
            control: self.control,
            alt: self.alt,
            super_key: self.meta,
        }
    }
}

/// Browser pointer event in CSS pixels relative to the canvas top-left corner.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct WebPointerEvent {
    pub kind: WebPointerKind,
    pub css_x: f64,
    pub css_y: f64,
    pub button: i16,
    pub modifiers: WebModifiers,
}

/// Browser wheel event in CSS pixels relative to the canvas top-left corner.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct WebWheelEvent {
    pub css_x: f64,
    pub css_y: f64,
    pub delta_y: f64,
    pub delta_mode: u32,
}

/// Browser keyboard event carrying W3C `KeyboardEvent.code` and optional `KeyboardEvent.key`.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct WebKeyEvent {
    pub code: String,
    pub key: String,
    pub pressed: bool,
    pub modifiers: WebModifiers,
}

/// Browser canvas resize event carrying CSS dimensions and device pixel ratio.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct WebResizeEvent {
    pub css_width: f64,
    pub css_height: f64,
    pub device_pixel_ratio: f64,
}

/// Translate a W3C `KeyboardEvent.code` string into [`ViewportPhysicalKey`].
/// Example: `assert_eq!(translate_dom_code("KeyF"), Some(ViewportPhysicalKey::F));`.
pub fn translate_dom_code(code: &str) -> Option<ViewportPhysicalKey> {
    use ViewportPhysicalKey as K;
    if let Some(key) = translate_letter_or_digit_code(code) {
        return Some(key);
    }
    if let Some(key) = translate_function_or_keypad_code(code) {
        return Some(key);
    }
    match code {
        "Tab" => Some(K::Tab),
        "ArrowLeft" => Some(K::LeftArrow),
        "ArrowRight" => Some(K::RightArrow),
        "ArrowUp" => Some(K::UpArrow),
        "ArrowDown" => Some(K::DownArrow),
        "PageUp" => Some(K::PageUp),
        "PageDown" => Some(K::PageDown),
        "Home" => Some(K::Home),
        "End" => Some(K::End),
        "Insert" => Some(K::Insert),
        "Delete" => Some(K::Delete),
        "Backspace" => Some(K::Backspace),
        "Space" => Some(K::Space),
        "Enter" => Some(K::Enter),
        "Escape" => Some(K::Escape),
        "ControlLeft" => Some(K::LeftCtrl),
        "ShiftLeft" => Some(K::LeftShift),
        "AltLeft" => Some(K::LeftAlt),
        "MetaLeft" | "OSLeft" => Some(K::LeftSuper),
        "ControlRight" => Some(K::RightCtrl),
        "ShiftRight" => Some(K::RightShift),
        "AltRight" => Some(K::RightAlt),
        "MetaRight" | "OSRight" => Some(K::RightSuper),
        "ContextMenu" => Some(K::Menu),
        "Quote" => Some(K::Apostrophe),
        "Comma" => Some(K::Comma),
        "Minus" => Some(K::Minus),
        "Period" => Some(K::Period),
        "Slash" => Some(K::Slash),
        "Semicolon" => Some(K::Semicolon),
        "Equal" => Some(K::Equal),
        "BracketLeft" => Some(K::LeftBracket),
        "Backslash" => Some(K::Backslash),
        "BracketRight" => Some(K::RightBracket),
        "Backquote" => Some(K::GraveAccent),
        "CapsLock" => Some(K::CapsLock),
        "ScrollLock" => Some(K::ScrollLock),
        "NumLock" => Some(K::NumLock),
        "PrintScreen" => Some(K::PrintScreen),
        "Pause" => Some(K::Pause),
        _ => None,
    }
}

fn translate_letter_or_digit_code(code: &str) -> Option<ViewportPhysicalKey> {
    use ViewportPhysicalKey as K;
    match code {
        "Digit0" => Some(K::Alpha0),
        "Digit1" => Some(K::Alpha1),
        "Digit2" => Some(K::Alpha2),
        "Digit3" => Some(K::Alpha3),
        "Digit4" => Some(K::Alpha4),
        "Digit5" => Some(K::Alpha5),
        "Digit6" => Some(K::Alpha6),
        "Digit7" => Some(K::Alpha7),
        "Digit8" => Some(K::Alpha8),
        "Digit9" => Some(K::Alpha9),
        "KeyA" => Some(K::A),
        "KeyB" => Some(K::B),
        "KeyC" => Some(K::C),
        "KeyD" => Some(K::D),
        "KeyE" => Some(K::E),
        "KeyF" => Some(K::F),
        "KeyG" => Some(K::G),
        "KeyH" => Some(K::H),
        "KeyI" => Some(K::I),
        "KeyJ" => Some(K::J),
        "KeyK" => Some(K::K),
        "KeyL" => Some(K::L),
        "KeyM" => Some(K::M),
        "KeyN" => Some(K::N),
        "KeyO" => Some(K::O),
        "KeyP" => Some(K::P),
        "KeyQ" => Some(K::Q),
        "KeyR" => Some(K::R),
        "KeyS" => Some(K::S),
        "KeyT" => Some(K::T),
        "KeyU" => Some(K::U),
        "KeyV" => Some(K::V),
        "KeyW" => Some(K::W),
        "KeyX" => Some(K::X),
        "KeyY" => Some(K::Y),
        "KeyZ" => Some(K::Z),
        _ => None,
    }
}

fn translate_function_or_keypad_code(code: &str) -> Option<ViewportPhysicalKey> {
    use ViewportPhysicalKey as K;
    match code {
        "F1" => Some(K::F1),
        "F2" => Some(K::F2),
        "F3" => Some(K::F3),
        "F4" => Some(K::F4),
        "F5" => Some(K::F5),
        "F6" => Some(K::F6),
        "F7" => Some(K::F7),
        "F8" => Some(K::F8),
        "F9" => Some(K::F9),
        "F10" => Some(K::F10),
        "F11" => Some(K::F11),
        "F12" => Some(K::F12),
        "Numpad0" => Some(K::Keypad0),
        "Numpad1" => Some(K::Keypad1),
        "Numpad2" => Some(K::Keypad2),
        "Numpad3" => Some(K::Keypad3),
        "Numpad4" => Some(K::Keypad4),
        "Numpad5" => Some(K::Keypad5),
        "Numpad6" => Some(K::Keypad6),
        "Numpad7" => Some(K::Keypad7),
        "Numpad8" => Some(K::Keypad8),
        "Numpad9" => Some(K::Keypad9),
        "NumpadDecimal" => Some(K::KeypadDecimal),
        "NumpadDivide" => Some(K::KeypadDivide),
        "NumpadMultiply" => Some(K::KeypadMultiply),
        "NumpadSubtract" => Some(K::KeypadSubtract),
        "NumpadAdd" => Some(K::KeypadAdd),
        "NumpadEnter" => Some(K::KeypadEnter),
        "NumpadEqual" => Some(K::KeypadEqual),
        _ => None,
    }
}

/// Translate a browser pointer event using the active device pixel ratio.
/// Example: `translate_web_pointer(event, 2.0)`.
pub fn translate_web_pointer(event: WebPointerEvent, dpr: f64) -> Option<ViewportEvent> {
    let position = scaled_pixel(event.css_x, event.css_y, dpr)?;
    let modifiers = event.modifiers.into_viewport();
    match event.kind {
        WebPointerKind::Move => Some(ViewportEvent::Input(ViewportInput::Motion {
            position,
            modifiers,
        })),
        WebPointerKind::Down | WebPointerKind::Up => {
            let button = match event.button {
                0 => ViewportButton::Left,
                1 => ViewportButton::Middle,
                2 => ViewportButton::Right,
                _ => return None,
            };
            Some(ViewportEvent::Input(ViewportInput::Button {
                button,
                pressed: event.kind == WebPointerKind::Down,
                position,
                modifiers,
            }))
        }
    }
}

/// Translate a browser wheel event into discrete zoom notches at physical pixel coordinates.
/// Example: `translate_web_wheel(event, 2.0)`.
pub fn translate_web_wheel(event: WebWheelEvent, dpr: f64) -> Option<ViewportEvent> {
    if !event.delta_y.is_finite() || event.delta_y == 0.0 {
        return None;
    }
    let position = scaled_pixel(event.css_x, event.css_y, dpr)?;
    let normalized = match event.delta_mode {
        1 => -event.delta_y,
        2 => -event.delta_y * 3.0,
        _ => -event.delta_y / 100.0,
    };
    let notches = if normalized > 0.0 {
        normalized.round().max(1.0) as i32
    } else {
        normalized.round().min(-1.0) as i32
    };
    Some(ViewportEvent::Input(ViewportInput::Wheel {
        notches,
        position,
    }))
}

/// Translate a browser keyboard event into physical key and optional Unicode text events.
/// Example: `let events = translate_web_key(&event);`.
pub fn translate_web_key(event: &WebKeyEvent) -> Vec<ViewportEvent> {
    let modifiers = event.modifiers.into_viewport();
    let mut output = Vec::new();
    if let Some(physical) = translate_dom_code(&event.code) {
        let command = shortcut_command(physical, modifiers);
        output.push(ViewportEvent::Input(ViewportInput::PhysicalKey {
            key: physical,
            layout_key: physical,
            command,
            pressed: event.pressed,
            modifiers,
        }));
    }
    if event.pressed
        && !modifiers.control
        && !modifiers.alt
        && !modifiers.super_key
        && event.key.chars().count() == 1
        && let Some(ch) = event.key.chars().next()
        && !ch.is_control()
    {
        output.push(ViewportEvent::Input(ViewportInput::Text(ch)));
    }
    output
}

/// Compute validated physical framebuffer dimensions from a browser resize event.
/// Example: `let extent = translate_web_resize(event)?;`.
pub fn translate_web_resize(event: WebResizeEvent) -> Result<[u32; 2], String> {
    let ratio = valid_dpr(event.device_pixel_ratio);
    if !event.css_width.is_finite()
        || !event.css_height.is_finite()
        || event.css_width <= 0.0
        || event.css_height <= 0.0
    {
        return Err(format!(
            "web resize {event:?}: expected finite positive CSS dimensions"
        ));
    }
    let width = (event.css_width * ratio).round().max(1.0);
    let height = (event.css_height * ratio).round().max(1.0);
    if width > f64::from(i32::MAX) || height > f64::from(i32::MAX) {
        return Err(format!(
            "web resize {event:?}: expected physical dimensions within signed 32-bit range"
        ));
    }
    Ok([width as u32, height as u32])
}

fn shortcut_command(key: ViewportPhysicalKey, mods: ViewportModifiers) -> Option<ViewportKey> {
    let primary = mods.control || mods.super_key;
    match (key, primary, mods.shift) {
        (ViewportPhysicalKey::Escape, false, false) => Some(ViewportKey::Cancel),
        (ViewportPhysicalKey::Home, false, false) => Some(ViewportKey::Home),
        (ViewportPhysicalKey::F2, false, false) => Some(ViewportKey::Pan),
        (ViewportPhysicalKey::F3, false, false) => Some(ViewportKey::Zoom),
        (ViewportPhysicalKey::F4, false, false) => Some(ViewportKey::Orbit),
        (ViewportPhysicalKey::F5, false, false) => Some(ViewportKey::Previous),
        (ViewportPhysicalKey::S, true, false) => Some(ViewportKey::Save),
        (ViewportPhysicalKey::Z, true, false) => Some(ViewportKey::Undo),
        (ViewportPhysicalKey::Z, true, true) | (ViewportPhysicalKey::Y, true, false) => {
            Some(ViewportKey::Redo)
        }
        _ => None,
    }
}

fn valid_dpr(dpr: f64) -> f64 {
    if dpr.is_finite() && dpr > 0.0 {
        dpr
    } else {
        1.0
    }
}

fn scaled_pixel(css_x: f64, css_y: f64, dpr: f64) -> Option<[i32; 2]> {
    if !css_x.is_finite() || !css_y.is_finite() {
        return None;
    }
    let scale = valid_dpr(dpr);
    let x = (css_x * scale).round();
    let y = (css_y * scale).round();
    if !(f64::from(i32::MIN)..=f64::from(i32::MAX)).contains(&x)
        || !(f64::from(i32::MIN)..=f64::from(i32::MAX)).contains(&y)
    {
        return None;
    }
    Some([x as i32, y as i32])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn translates_pointer_wheel_resize_and_keyboard_events() {
        let down = translate_web_pointer(
            WebPointerEvent {
                kind: WebPointerKind::Down,
                css_x: 10.5,
                css_y: 20.0,
                button: 0,
                modifiers: WebModifiers {
                    shift: true,
                    ..Default::default()
                },
            },
            2.0,
        )
        .unwrap();
        assert_eq!(
            down,
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Left,
                pressed: true,
                position: [21, 40],
                modifiers: ViewportModifiers {
                    shift: true,
                    ..Default::default()
                },
            })
        );
        assert!(
            translate_web_pointer(
                WebPointerEvent {
                    kind: WebPointerKind::Up,
                    css_x: 1.0,
                    css_y: 1.0,
                    button: 4,
                    modifiers: Default::default(),
                },
                1.0
            )
            .is_none()
        );
        assert!(matches!(
            translate_web_pointer(
                WebPointerEvent {
                    kind: WebPointerKind::Move,
                    css_x: 5.0,
                    css_y: 6.0,
                    button: 0,
                    modifiers: Default::default(),
                },
                1.0
            ),
            Some(ViewportEvent::Input(ViewportInput::Motion {
                position: [5, 6],
                ..
            }))
        ));
        let wheel = translate_web_wheel(
            WebWheelEvent {
                css_x: 50.0,
                css_y: 25.0,
                delta_y: -120.0,
                delta_mode: 0,
            },
            2.0,
        )
        .unwrap();
        assert_eq!(
            wheel,
            ViewportEvent::Input(ViewportInput::Wheel {
                notches: 1,
                position: [100, 50],
            })
        );
        assert!(
            translate_web_wheel(
                WebWheelEvent {
                    css_x: 0.0,
                    css_y: 0.0,
                    delta_y: 0.0,
                    delta_mode: 0,
                },
                1.0
            )
            .is_none()
        );
        assert_eq!(
            translate_web_resize(WebResizeEvent {
                css_width: 640.0,
                css_height: 360.0,
                device_pixel_ratio: 1.5,
            })
            .unwrap(),
            [960, 540]
        );
        assert!(
            translate_web_resize(WebResizeEvent {
                css_width: 0.0,
                css_height: 100.0,
                device_pixel_ratio: 1.0,
            })
            .is_err()
        );
        let typed = translate_web_key(&WebKeyEvent {
            code: "KeyA".into(),
            key: "a".into(),
            pressed: true,
            modifiers: Default::default(),
        });
        assert_eq!(typed.len(), 2);
        assert_eq!(typed[1], ViewportEvent::Input(ViewportInput::Text('a')));
        let undo = translate_web_key(&WebKeyEvent {
            code: "KeyZ".into(),
            key: "z".into(),
            pressed: true,
            modifiers: WebModifiers {
                control: true,
                ..Default::default()
            },
        });
        assert_eq!(undo.len(), 1);
        assert_eq!(translate_dom_code("F4"), Some(ViewportPhysicalKey::F4));
        assert_eq!(
            translate_dom_code("NumpadEnter"),
            Some(ViewportPhysicalKey::KeypadEnter)
        );
        assert_eq!(
            translate_dom_code("BracketLeft"),
            Some(ViewportPhysicalKey::LeftBracket)
        );
        assert_eq!(translate_dom_code("UnknownKey"), None);
    }
}
