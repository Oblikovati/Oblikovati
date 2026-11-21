// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{ViewportButton, ViewportInput, ViewportKey, ViewportModifiers, ViewportPhysicalKey};
use windows_sys::Win32::{
    Foundation::POINT, Graphics::Gdi::ScreenToClient, UI::Input::KeyboardAndMouse::*,
};

// Low-word flags in the mouse messages for left, right and middle.
const MOUSE_BUTTONS: usize = 0x0001 | 0x0002 | 0x0010;

pub(super) fn dispatch(
    events: &WindowEvents,
    hwnd: HWND,
    msg: u32,
    wparam: WPARAM,
    lparam: LPARAM,
) -> Option<LRESULT> {
    let position = [lparam as i16 as i32, (lparam >> 16) as i16 as i32];
    let event = match msg {
        WM_MOUSEMOVE => Some(ViewportInput::Motion {
            position,
            modifiers: modifiers(),
        }),
        WM_LBUTTONDOWN | WM_LBUTTONUP | WM_MBUTTONDOWN | WM_MBUTTONUP | WM_RBUTTONDOWN
        | WM_RBUTTONUP => return Some(button(events, hwnd, msg, wparam, position)),
        WM_MOUSEWHEEL => wheel(events, hwnd, wparam, position),
        WM_CAPTURECHANGED | WM_CANCELMODE => {
            if !events.releasing_capture.get() {
                events.push(ViewportEvent::Focus(false));
            }
            return None;
        }
        WM_KEYDOWN | WM_KEYUP | WM_SYSKEYDOWN | WM_SYSKEYUP => keyboard(msg, wparam as u16, lparam),
        WM_CHAR | WM_SYSCHAR => text(events, wparam as u16),
        _ => return None,
    };
    if let Some(event) = event {
        events.push(ViewportEvent::Input(event));
    }
    // Preserve native system shortcuts such as Alt+F4 and Alt+Space.
    if matches!(msg, WM_SYSKEYDOWN | WM_SYSKEYUP) {
        None
    } else {
        Some(0)
    }
}

fn button(
    events: &WindowEvents,
    hwnd: HWND,
    msg: u32,
    wparam: WPARAM,
    position: [i32; 2],
) -> LRESULT {
    let button = match msg {
        WM_LBUTTONDOWN | WM_LBUTTONUP => ViewportButton::Left,
        WM_MBUTTONDOWN | WM_MBUTTONUP => ViewportButton::Middle,
        _ => ViewportButton::Right,
    };
    let pressed = matches!(msg, WM_LBUTTONDOWN | WM_MBUTTONDOWN | WM_RBUTTONDOWN);
    events.push(ViewportEvent::Input(ViewportInput::Button {
        button,
        pressed,
        position,
        modifiers: modifiers(),
    }));
    if pressed {
        unsafe {
            SetCapture(hwnd);
        }
    } else if wparam & MOUSE_BUTTONS == 0 && unsafe { GetCapture() } == hwnd {
        // ReleaseCapture synchronously dispatches WM_CAPTURECHANGED to this window.
        events.releasing_capture.set(true);
        unsafe {
            ReleaseCapture();
        }
        events.releasing_capture.set(false);
    }
    0
}

fn wheel(
    events: &WindowEvents,
    hwnd: HWND,
    wparam: WPARAM,
    position: [i32; 2],
) -> Option<ViewportInput> {
    let steps = events.wheel.get() + (wparam >> 16) as i16 as i32;
    events.wheel.set(steps % 120);
    let mut point = POINT {
        x: position[0],
        y: position[1],
    };
    if steps.abs() < 120 || unsafe { ScreenToClient(hwnd, &mut point) } == 0 {
        return None;
    }
    Some(ViewportInput::Wheel {
        notches: steps / 120,
        position: [point.x, point.y],
    })
}

fn modifiers() -> ViewportModifiers {
    fn down(key: u16) -> bool {
        unsafe { GetKeyState(i32::from(key)) < 0 }
    }
    ViewportModifiers {
        shift: down(VK_SHIFT),
        control: down(VK_CONTROL),
        alt: down(VK_MENU),
        super_key: down(VK_LWIN) || down(VK_RWIN),
    }
}

fn keyboard(msg: u32, vk: u16, flags: LPARAM) -> Option<ViewportInput> {
    let scan = ((flags >> 16) & 0xff) as u32 | if flags & (1 << 24) != 0 { 0xe000 } else { 0 };
    let vk = match vk {
        VK_CONTROL | VK_MENU | VK_SHIFT => unsafe {
            MapVirtualKeyW(scan, MAPVK_VSC_TO_VK_EX) as u16
        },
        vk => vk,
    };
    let layout_key = virtual_key(vk, scan)?;
    // Translate physical positions using the standard US scan-code table.
    let physical = physical_key(scan).unwrap_or(layout_key);
    let modifiers = modifiers();
    Some(ViewportInput::PhysicalKey {
        key: physical,
        layout_key,
        command: command(layout_key, modifiers),
        pressed: matches!(msg, WM_KEYDOWN | WM_SYSKEYDOWN),
        modifiers,
    })
}

fn command(key: ViewportPhysicalKey, mods: ViewportModifiers) -> Option<ViewportKey> {
    use ViewportPhysicalKey::*;
    let ctrl = mods.control && !mods.alt && !mods.super_key;
    Some(match key {
        S if ctrl && !mods.shift => ViewportKey::Save,
        Z if ctrl && !mods.shift => ViewportKey::Undo,
        Y if ctrl && !mods.shift => ViewportKey::Redo,
        Z if ctrl && mods.shift => ViewportKey::Redo,
        F2 => ViewportKey::Pan,
        F3 => ViewportKey::Zoom,
        F4 => ViewportKey::Orbit,
        F5 => ViewportKey::Previous,
        F6 => ViewportKey::Home,
        Escape => ViewportKey::Cancel,
        _ => return None,
    })
}

fn text(events: &WindowEvents, unit: u16) -> Option<ViewportInput> {
    let lead = events.surrogate.take();
    if (0xd800..=0xdbff).contains(&unit) {
        events.surrogate.set(Some(unit));
        return None;
    }
    let code = if let Some(lead) = lead.filter(|_| (0xdc00..=0xdfff).contains(&unit)) {
        0x10000 + ((u32::from(lead) - 0xd800) << 10) + u32::from(unit) - 0xdc00
    } else {
        u32::from(unit)
    };
    char::from_u32(code)
        .filter(|ch| !ch.is_control())
        .map(ViewportInput::Text)
}

fn physical_key(scan: u32) -> Option<ViewportPhysicalKey> {
    use ViewportPhysicalKey::*;
    let (start, row): (u32, &[ViewportPhysicalKey]) = match scan {
        0x02..=0x0d => (
            0x02,
            &[
                Alpha1, Alpha2, Alpha3, Alpha4, Alpha5, Alpha6, Alpha7, Alpha8, Alpha9, Alpha0,
                Minus, Equal,
            ],
        ),
        0x10..=0x1b => (
            0x10,
            &[Q, W, E, R, T, Y, U, I, O, P, LeftBracket, RightBracket],
        ),
        0x1e..=0x29 => (
            0x1e,
            &[
                A,
                S,
                D,
                F,
                G,
                H,
                J,
                K,
                L,
                Semicolon,
                Apostrophe,
                GraveAccent,
            ],
        ),
        0x2b..=0x35 => (
            0x2b,
            &[Backslash, Z, X, C, V, B, N, M, Comma, Period, Slash],
        ),
        _ => return None,
    };
    Some(row[(scan - start) as usize])
}

fn virtual_key(vk: u16, scan: u32) -> Option<ViewportPhysicalKey> {
    use ViewportPhysicalKey::*;
    Some(match vk {
        VK_RETURN if scan == 0xe01c => KeypadEnter,
        0x30 => Alpha0,
        0x31 => Alpha1,
        0x32 => Alpha2,
        0x33 => Alpha3,
        0x34 => Alpha4,
        0x35 => Alpha5,
        0x36 => Alpha6,
        0x37 => Alpha7,
        0x38 => Alpha8,
        0x39 => Alpha9,
        0x41 => A,
        0x42 => B,
        0x43 => C,
        0x44 => D,
        0x45 => E,
        0x46 => F,
        0x47 => G,
        0x48 => H,
        0x49 => I,
        0x4a => J,
        0x4b => K,
        0x4c => L,
        0x4d => M,
        0x4e => N,
        0x4f => O,
        0x50 => P,
        0x51 => Q,
        0x52 => R,
        0x53 => S,
        0x54 => T,
        0x55 => U,
        0x56 => V,
        0x57 => W,
        0x58 => X,
        0x59 => Y,
        0x5a => Z,
        VK_F1 => F1,
        VK_F2 => F2,
        VK_F3 => F3,
        VK_F4 => F4,
        VK_F5 => F5,
        VK_F6 => F6,
        VK_F7 => F7,
        VK_F8 => F8,
        VK_F9 => F9,
        VK_F10 => F10,
        VK_F11 => F11,
        VK_F12 => F12,
        VK_NUMPAD0 => Keypad0,
        VK_NUMPAD1 => Keypad1,
        VK_NUMPAD2 => Keypad2,
        VK_NUMPAD3 => Keypad3,
        VK_NUMPAD4 => Keypad4,
        VK_NUMPAD5 => Keypad5,
        VK_NUMPAD6 => Keypad6,
        VK_NUMPAD7 => Keypad7,
        VK_NUMPAD8 => Keypad8,
        VK_NUMPAD9 => Keypad9,
        VK_ESCAPE => Escape,
        VK_TAB => Tab,
        VK_BACK => Backspace,
        VK_RETURN => Enter,
        VK_SPACE => Space,
        VK_LEFT => LeftArrow,
        VK_RIGHT => RightArrow,
        VK_UP => UpArrow,
        VK_DOWN => DownArrow,
        VK_PRIOR => PageUp,
        VK_NEXT => PageDown,
        VK_HOME => Home,
        VK_END => End,
        VK_INSERT => Insert,
        VK_DELETE => Delete,
        VK_LCONTROL => LeftCtrl,
        VK_RCONTROL => RightCtrl,
        VK_LSHIFT => LeftShift,
        VK_RSHIFT => RightShift,
        VK_LMENU => LeftAlt,
        VK_RMENU => RightAlt,
        VK_LWIN => LeftSuper,
        VK_RWIN => RightSuper,
        VK_APPS => Menu,
        VK_CAPITAL => CapsLock,
        VK_SCROLL => ScrollLock,
        VK_NUMLOCK => NumLock,
        VK_SNAPSHOT => PrintScreen,
        VK_PAUSE => Pause,
        VK_DECIMAL => KeypadDecimal,
        VK_DIVIDE => KeypadDivide,
        VK_MULTIPLY => KeypadMultiply,
        VK_SUBTRACT => KeypadSubtract,
        VK_ADD => KeypadAdd,
        VK_OEM_1 => Semicolon,
        VK_OEM_PLUS => Equal,
        VK_OEM_COMMA => Comma,
        VK_OEM_MINUS => Minus,
        VK_OEM_PERIOD => Period,
        VK_OEM_2 => Slash,
        VK_OEM_3 => GraveAccent,
        VK_OEM_4 => LeftBracket,
        VK_OEM_5 => Backslash,
        VK_OEM_6 => RightBracket,
        VK_OEM_7 => Apostrophe,
        _ => return None,
    })
}
