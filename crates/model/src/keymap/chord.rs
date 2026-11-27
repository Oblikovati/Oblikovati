// SPDX-License-Identifier: GPL-2.0-only
//! Canonical keyboard chord parsing for the public keymap contract.
use oblikovati_api::types::KeyChord;

/// Parse the public chord format; example: `parse_chord("control+shift+e")?`.
pub fn parse_chord(input: &str) -> Result<KeyChord, String> {
    if input.trim().is_empty() {
        return Ok(KeyChord::default());
    }
    let mut parts: Vec<_> = input.split('+').collect();
    let key = parts.pop().unwrap_or_default();
    if key.is_empty() {
        return Err(format!(
            "key chord {input:?}: expected [Ctrl+][Alt+][Shift+]Key"
        ));
    }
    let mut chord = KeyChord {
        key: canonical_key(key),
        ..Default::default()
    };
    for modifier in parts {
        match modifier.to_ascii_lowercase().as_str() {
            "ctrl" | "control" => chord.ctrl = true,
            "alt" | "option" => chord.alt = true,
            "shift" => chord.shift = true,
            _ => {
                return Err(format!(
                    "key chord {input:?}, modifier {modifier:?}: expected Ctrl, Alt or Shift"
                ));
            }
        }
    }
    Ok(chord)
}

fn canonical_key(key: &str) -> String {
    if key.chars().count() == 1 {
        key.to_uppercase()
    } else {
        key.to_owned()
    }
}

/// Format the public chord canonically; example: `format_chord(&chord) == "Ctrl+Z"`.
pub fn format_chord(chord: &KeyChord) -> String {
    if chord.key.is_empty() {
        return String::new();
    }
    let mut result = String::new();
    for (enabled, name) in [
        (chord.ctrl, "Ctrl+"),
        (chord.alt, "Alt+"),
        (chord.shift, "Shift+"),
    ] {
        if enabled {
            result.push_str(name);
        }
    }
    result.push_str(&canonical_key(&chord.key));
    result
}

pub(super) fn reserved_bare(chord: &KeyChord) -> bool {
    !chord.ctrl
        && !chord.alt
        && !chord.shift
        && chord.key.chars().count() == 1
        && chord.key.chars().next().is_some_and(char::is_alphanumeric)
}

pub(super) fn comparison(chord: &str) -> String {
    let (prefix, key) = chord.rsplit_once('+').unwrap_or(("", chord));
    let key = match key {
        "Apostrophe" => "'",
        "Comma" => ",",
        "Minus" => "-",
        "Period" => ".",
        "Slash" => "/",
        "Semicolon" => ";",
        "Equal" => "=",
        "LeftBracket" => "[",
        "Backslash" => "\\",
        "RightBracket" => "]",
        "GraveAccent" => "`",
        _ => key,
    };
    if prefix.is_empty() {
        key.to_owned()
    } else {
        format!("{prefix}+{key}")
    }
}

pub(super) fn require_supported(chord: &KeyChord) -> Result<(), String> {
    if chord.key.is_empty()
        || super::PhysicalKey::ALL
            .iter()
            .any(|key| comparison(&key.chord_token()) == comparison(&chord.key))
    {
        return Ok(());
    }
    Err(format!(
        "key chord {:?}: expected a key supported by the desktop",
        format_chord(chord)
    ))
}
