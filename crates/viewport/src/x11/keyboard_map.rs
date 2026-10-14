// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::ViewportPhysicalKey;
use std::collections::BTreeMap;
use std::ptr::NonNull;
// x11-dl 2.21.0 omits these XKB.h protocol constants.
const CORE_KEYBOARD: u32 = 0x0100;
const KEY_NAMES_MASK: u32 = 1 << 9;
pub(super) struct NativeKeymap(BTreeMap<u32, ViewportPhysicalKey>);
struct OwnedXkbMap<'a> {
    api: &'a xlib::Xlib,
    pointer: NonNull<xlib::_XkbDesc>,
}
impl Drop for OwnedXkbMap<'_> {
    fn drop(&mut self) {
        unsafe { (self.api.XkbFreeKeyboard)(self.pointer.as_ptr(), 0, 1) };
    }
}
impl NativeKeymap {
    pub(super) fn load(api: &xlib::Xlib, display: *mut xlib::Display) -> Result<Self, String> {
        let pointer = unsafe { (api.XkbGetMap)(display, 0, CORE_KEYBOARD) };
        let pointer =
            NonNull::new(pointer).ok_or("XKB core keyboard: expected an owned keyboard map")?;
        let keyboard = OwnedXkbMap { api, pointer };
        let status = unsafe { (api.XkbGetNames)(display, KEY_NAMES_MASK, pointer.as_ptr()) };
        if status != xlib::Success as i32 {
            return Err(format!("XKB names status {status}: expected Success"));
        }
        copy_physical_keymap(&keyboard)
    }
    pub(super) fn key(&self, code: u32) -> Option<ViewportPhysicalKey> {
        self.0.get(&code).copied()
    }
}
fn copy_physical_keymap(keyboard: &OwnedXkbMap<'_>) -> Result<NativeKeymap, String> {
    let descriptor = unsafe { keyboard.pointer.as_ref() };
    let names =
        unsafe { descriptor.names.as_ref() }.ok_or("XKB names: expected a live key-name table")?;
    if names.keys.is_null() || descriptor.min_key_code > descriptor.max_key_code {
        return Err(format!(
            "XKB key bounds {}..={}: expected an owned ordered key-name array",
            descriptor.min_key_code, descriptor.max_key_code
        ));
    }
    let mut copied = BTreeMap::new();
    for code in descriptor.min_key_code..=descriptor.max_key_code {
        // XKB names.keys is indexed by the descriptor's native keycode bounds.
        let name = unsafe { (*names.keys.add(usize::from(code))).name }.map(|byte| byte as u8);
        if let Some(key) = physical_key_name(name) {
            copied.insert(u32::from(code), key);
        }
    }
    Ok(NativeKeymap(copied))
}
const XKB_POSITION_KEYS: &[([u8; 4], ViewportPhysicalKey)] = &[
    (*b"AE01", ViewportPhysicalKey::Alpha1),
    (*b"AE02", ViewportPhysicalKey::Alpha2),
    (*b"AE03", ViewportPhysicalKey::Alpha3),
    (*b"AE04", ViewportPhysicalKey::Alpha4),
    (*b"AE05", ViewportPhysicalKey::Alpha5),
    (*b"AE06", ViewportPhysicalKey::Alpha6),
    (*b"AE07", ViewportPhysicalKey::Alpha7),
    (*b"AE08", ViewportPhysicalKey::Alpha8),
    (*b"AE09", ViewportPhysicalKey::Alpha9),
    (*b"AE10", ViewportPhysicalKey::Alpha0),
    (*b"AE11", ViewportPhysicalKey::Minus),
    (*b"AE12", ViewportPhysicalKey::Equal),
    (*b"AD01", ViewportPhysicalKey::Q),
    (*b"AD02", ViewportPhysicalKey::W),
    (*b"AD03", ViewportPhysicalKey::E),
    (*b"AD04", ViewportPhysicalKey::R),
    (*b"AD05", ViewportPhysicalKey::T),
    (*b"AD06", ViewportPhysicalKey::Y),
    (*b"AD07", ViewportPhysicalKey::U),
    (*b"AD08", ViewportPhysicalKey::I),
    (*b"AD09", ViewportPhysicalKey::O),
    (*b"AD10", ViewportPhysicalKey::P),
    (*b"AD11", ViewportPhysicalKey::LeftBracket),
    (*b"AD12", ViewportPhysicalKey::RightBracket),
    (*b"AC01", ViewportPhysicalKey::A),
    (*b"AC02", ViewportPhysicalKey::S),
    (*b"AC03", ViewportPhysicalKey::D),
    (*b"AC04", ViewportPhysicalKey::F),
    (*b"AC05", ViewportPhysicalKey::G),
    (*b"AC06", ViewportPhysicalKey::H),
    (*b"AC07", ViewportPhysicalKey::J),
    (*b"AC08", ViewportPhysicalKey::K),
    (*b"AC09", ViewportPhysicalKey::L),
    (*b"AC10", ViewportPhysicalKey::Semicolon),
    (*b"AC11", ViewportPhysicalKey::Apostrophe),
    (*b"AB01", ViewportPhysicalKey::Z),
    (*b"AB02", ViewportPhysicalKey::X),
    (*b"AB03", ViewportPhysicalKey::C),
    (*b"AB04", ViewportPhysicalKey::V),
    (*b"AB05", ViewportPhysicalKey::B),
    (*b"AB06", ViewportPhysicalKey::N),
    (*b"AB07", ViewportPhysicalKey::M),
    (*b"AB08", ViewportPhysicalKey::Comma),
    (*b"AB09", ViewportPhysicalKey::Period),
    (*b"AB10", ViewportPhysicalKey::Slash),
    (*b"FK01", ViewportPhysicalKey::F1),
    (*b"FK02", ViewportPhysicalKey::F2),
    (*b"FK03", ViewportPhysicalKey::F3),
    (*b"FK04", ViewportPhysicalKey::F4),
    (*b"FK05", ViewportPhysicalKey::F5),
    (*b"FK06", ViewportPhysicalKey::F6),
    (*b"FK07", ViewportPhysicalKey::F7),
    (*b"FK08", ViewportPhysicalKey::F8),
    (*b"FK09", ViewportPhysicalKey::F9),
    (*b"FK10", ViewportPhysicalKey::F10),
    (*b"FK11", ViewportPhysicalKey::F11),
    (*b"FK12", ViewportPhysicalKey::F12),
    (*b"KP0\0", ViewportPhysicalKey::Keypad0),
    (*b"KP1\0", ViewportPhysicalKey::Keypad1),
    (*b"KP2\0", ViewportPhysicalKey::Keypad2),
    (*b"KP3\0", ViewportPhysicalKey::Keypad3),
    (*b"KP4\0", ViewportPhysicalKey::Keypad4),
    (*b"KP5\0", ViewportPhysicalKey::Keypad5),
    (*b"KP6\0", ViewportPhysicalKey::Keypad6),
    (*b"KP7\0", ViewportPhysicalKey::Keypad7),
    (*b"KP8\0", ViewportPhysicalKey::Keypad8),
    (*b"KP9\0", ViewportPhysicalKey::Keypad9),
    (*b"ESC\0", ViewportPhysicalKey::Escape),
    (*b"TAB\0", ViewportPhysicalKey::Tab),
    (*b"BKSP", ViewportPhysicalKey::Backspace),
    (*b"RTRN", ViewportPhysicalKey::Enter),
    (*b"SPCE", ViewportPhysicalKey::Space),
    (*b"LEFT", ViewportPhysicalKey::LeftArrow),
    (*b"RGHT", ViewportPhysicalKey::RightArrow),
    (*b"UP\0\0", ViewportPhysicalKey::UpArrow),
    (*b"DOWN", ViewportPhysicalKey::DownArrow),
    (*b"PGUP", ViewportPhysicalKey::PageUp),
    (*b"PGDN", ViewportPhysicalKey::PageDown),
    (*b"HOME", ViewportPhysicalKey::Home),
    (*b"END\0", ViewportPhysicalKey::End),
    (*b"INS\0", ViewportPhysicalKey::Insert),
    (*b"DELE", ViewportPhysicalKey::Delete),
    (*b"LCTL", ViewportPhysicalKey::LeftCtrl),
    (*b"RCTL", ViewportPhysicalKey::RightCtrl),
    (*b"LFSH", ViewportPhysicalKey::LeftShift),
    (*b"RTSH", ViewportPhysicalKey::RightShift),
    (*b"LALT", ViewportPhysicalKey::LeftAlt),
    (*b"RALT", ViewportPhysicalKey::RightAlt),
    (*b"LWIN", ViewportPhysicalKey::LeftSuper),
    (*b"RWIN", ViewportPhysicalKey::RightSuper),
    (*b"MENU", ViewportPhysicalKey::Menu),
    (*b"TLDE", ViewportPhysicalKey::GraveAccent),
    (*b"BKSL", ViewportPhysicalKey::Backslash),
    (*b"CAPS", ViewportPhysicalKey::CapsLock),
    (*b"SCLK", ViewportPhysicalKey::ScrollLock),
    (*b"NMLK", ViewportPhysicalKey::NumLock),
    (*b"PRSC", ViewportPhysicalKey::PrintScreen),
    (*b"PAUS", ViewportPhysicalKey::Pause),
    (*b"KPDL", ViewportPhysicalKey::KeypadDecimal),
    (*b"KPDV", ViewportPhysicalKey::KeypadDivide),
    (*b"KPMU", ViewportPhysicalKey::KeypadMultiply),
    (*b"KPSU", ViewportPhysicalKey::KeypadSubtract),
    (*b"KPAD", ViewportPhysicalKey::KeypadAdd),
    (*b"KPEN", ViewportPhysicalKey::KeypadEnter),
    (*b"KPEQ", ViewportPhysicalKey::KeypadEqual),
];
fn physical_key_name(name: [u8; 4]) -> Option<ViewportPhysicalKey> {
    XKB_POSITION_KEYS
        .iter()
        .find(|(position, _)| *position == name)
        .map(|(_, key)| *key)
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn xkb_positions_are_independent_from_shifted_character_spelling() {
        assert_eq!(physical_key_name(*b"AD01"), Some(ViewportPhysicalKey::Q));
        assert_eq!(physical_key_name(*b"AC01"), Some(ViewportPhysicalKey::A));
        assert_eq!(
            physical_key_name(*b"AE04"),
            Some(ViewportPhysicalKey::Alpha4)
        );
        assert_eq!(physical_key_name(*b"FK06"), Some(ViewportPhysicalKey::F6));
        assert_eq!(
            physical_key_name(*b"KPEN"),
            Some(ViewportPhysicalKey::KeypadEnter)
        );
        assert_eq!(physical_key_name(*b"NOPE"), None);
    }
}
