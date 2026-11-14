// SPDX-License-Identifier: GPL-2.0-only
use crate::ViewportPhysicalKey;
use ViewportPhysicalKey::*;
const ASCII_LAYOUT_KEYS: &[(u8, ViewportPhysicalKey)] = &[
    (b'a', A),
    (b'b', B),
    (b'c', C),
    (b'd', D),
    (b'e', E),
    (b'f', F),
    (b'g', G),
    (b'h', H),
    (b'i', I),
    (b'j', J),
    (b'k', K),
    (b'l', L),
    (b'm', M),
    (b'n', N),
    (b'o', O),
    (b'p', P),
    (b'q', Q),
    (b'r', R),
    (b's', S),
    (b't', T),
    (b'u', U),
    (b'v', V),
    (b'w', W),
    (b'x', X),
    (b'y', Y),
    (b'z', Z),
    (b'0', Alpha0),
    (b'1', Alpha1),
    (b'2', Alpha2),
    (b'3', Alpha3),
    (b'4', Alpha4),
    (b'5', Alpha5),
    (b'6', Alpha6),
    (b'7', Alpha7),
    (b'8', Alpha8),
    (b'9', Alpha9),
    (b'`', GraveAccent),
    (b'-', Minus),
    (b'=', Equal),
    (b'[', LeftBracket),
    (b']', RightBracket),
    (b'\\', Backslash),
    (b',', Comma),
    (b';', Semicolon),
    (b'\'', Apostrophe),
    (b'.', Period),
    (b'/', Slash),
];
const KEYPAD_POSITIONS: &[ViewportPhysicalKey] = &[
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
];
pub(super) fn named_layout_key(
    position: ViewportPhysicalKey,
    symbol: std::ffi::c_ulong,
) -> ViewportPhysicalKey {
    // Match Go imgui_impl_glfw.cpp letter shortcuts without translating keypad identities.
    if KEYPAD_POSITIONS.contains(&position) {
        return position;
    }
    let Ok(character) = u8::try_from(symbol) else {
        return position;
    };
    ASCII_LAYOUT_KEYS
        .iter()
        .find(|(letter, _)| *letter == character.to_ascii_lowercase())
        .map_or(position, |(_, key)| *key)
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn layout_letters_change_without_translating_keypad_or_unicode_only_symbols() {
        assert_eq!(named_layout_key(A, std::ffi::c_ulong::from(b'Q')), Q);
        assert_eq!(
            named_layout_key(Keypad1, std::ffi::c_ulong::from(b'1')),
            Keypad1
        );
        assert_eq!(named_layout_key(A, 0x0101_f642), A);
    }
}
