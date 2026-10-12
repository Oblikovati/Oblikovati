// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{ViewportButton, ViewportInput, ViewportKey, ViewportModifiers};
fn copied_modifiers(state: u32) -> ViewportModifiers {
    ViewportModifiers {
        shift: state & xlib::ShiftMask != 0,
        control: state & xlib::ControlMask != 0,
    }
}
fn copied_button(button: u32) -> Option<ViewportButton> {
    match button {
        1 => Some(ViewportButton::Left),
        2 => Some(ViewportButton::Middle),
        3 => Some(ViewportButton::Right),
        _ => None,
    }
}
pub(super) fn translate_pointer_key(
    api: &xlib::Xlib,
    owner: *mut xlib::Display,
    event: xlib::XEvent,
) -> Option<ViewportInput> {
    // Key lookup dereferences the event display; require the owning connection first.
    if owner.is_null() || unsafe { event.any.display } != owner {
        return None;
    }
    match event.get_type() {
        xlib::ButtonPress | xlib::ButtonRelease => button_input(unsafe { event.button }),
        xlib::MotionNotify => {
            let pointer = unsafe { event.motion };
            Some(ViewportInput::Motion {
                position: [pointer.x, pointer.y],
                modifiers: copied_modifiers(pointer.state),
            })
        }
        xlib::KeyPress | xlib::KeyRelease => key_input(api, unsafe { event.key }),
        _ => None,
    }
}
fn button_input(pointer: xlib::XButtonEvent) -> Option<ViewportInput> {
    let position = [pointer.x, pointer.y];
    let pressed = pointer.type_ == xlib::ButtonPress;
    if matches!(pointer.button, 4 | 5) {
        return pressed.then_some(ViewportInput::Wheel {
            notches: if pointer.button == 4 { 1 } else { -1 },
            position,
        });
    }
    Some(ViewportInput::Button {
        button: copied_button(pointer.button)?,
        pressed,
        position,
        modifiers: copied_modifiers(pointer.state),
    })
}
fn key_input(api: &xlib::Xlib, mut event: xlib::XKeyEvent) -> Option<ViewportInput> {
    let symbol = unsafe { (api.XLookupKeysym)(&mut event, 0) };
    let key = copied_key(symbol as u32, event.state)?;
    Some(ViewportInput::Key {
        key,
        pressed: event.type_ == xlib::KeyPress,
    })
}
#[cfg(test)]
#[path = "../../tests/x11_input.rs"]
mod tests;

fn control_only_chord(state: u32) -> bool {
    state & xlib::ControlMask != 0 && state & (xlib::ShiftMask | xlib::Mod1Mask) == 0
}

fn redo_shift_chord(state: u32) -> bool {
    state & (xlib::ControlMask | xlib::ShiftMask) == xlib::ControlMask | xlib::ShiftMask
        && state & xlib::Mod1Mask == 0
}

fn copied_key(symbol: u32, state: u32) -> Option<ViewportKey> {
    use x11_dl::keysym;
    let key = match symbol {
        keysym::XK_s | keysym::XK_S if control_only_chord(state) => ViewportKey::Save,
        keysym::XK_z | keysym::XK_Z if control_only_chord(state) => ViewportKey::Undo,
        keysym::XK_y | keysym::XK_Y if control_only_chord(state) => ViewportKey::Redo,
        keysym::XK_z | keysym::XK_Z if redo_shift_chord(state) => ViewportKey::Redo,
        keysym::XK_F2 => ViewportKey::Pan,
        keysym::XK_F3 => ViewportKey::Zoom,
        keysym::XK_F4 => ViewportKey::Orbit,
        keysym::XK_F6 => ViewportKey::Home,
        keysym::XK_F5 => ViewportKey::Previous,
        keysym::XK_Escape => ViewportKey::Cancel,
        _ => return None,
    };
    Some(key)
}
