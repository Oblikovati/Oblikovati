// SPDX-License-Identifier: GPL-2.0-only
use super::{Keymap, KeymapAction, format_chord, parse_chord};
use oblikovati_api::wire::KeymapExport;
use std::collections::BTreeMap;

fn sample_keymap() -> Keymap {
    Keymap::new(vec![
        KeymapAction::builtin("edit.undo", "Undo", "Ctrl+Z", &[]).unwrap(),
        KeymapAction::builtin("edit.redo", "Redo", "Ctrl+Y", &["Ctrl+Shift+Z"]).unwrap(),
        KeymapAction::command("draw.line", "Line", "").unwrap(),
    ])
    .unwrap()
}

#[test]
fn chord_parser_canonicalizes_and_rejects_invalid_input() {
    assert_eq!(
        format_chord(&parse_chord("option+control+shift+z").unwrap()),
        "Ctrl+Alt+Shift+Z"
    );
    assert_eq!(format_chord(&parse_chord("").unwrap()), "");
    assert!(parse_chord("Meta+Z").is_err());
    assert!(parse_chord("Ctrl+").is_err());
}
#[test]
fn punctuation_tokens_resolve_and_conflict_with_native_key_names() {
    let mut map = sample_keymap();
    map.set_chord("draw.line", "Ctrl+,").unwrap();
    assert_eq!(
        map.resolve_chord(&parse_chord("Ctrl+Comma").unwrap()),
        Some("draw.line")
    );
    assert!(map.set_chord("edit.undo", "Ctrl+Comma").is_err());
    assert_eq!(map.catalog()[2].chord, "Ctrl+,");
}

#[test]
fn desktop_key_inventory_admits_only_receivable_chords() {
    let mut map = sample_keymap();
    for key in super::PhysicalKey::ALL {
        let token = key.chord_token();
        map.set_chord("draw.line", &format!("Ctrl+Alt+{token}"))
            .unwrap_or_else(|error| panic!("key {token}: {error}"));
    }
    assert!(map.set_chord("draw.line", "Ctrl+NoSuchKey").is_err());
}

#[test]
fn conflicts_include_alternates_and_overrides_disable_them() {
    let mut map = sample_keymap();
    assert!(map.set_chord("draw.line", "Ctrl+Shift+Z").is_err());
    assert!(map.set_chord("draw.line", "L").is_err());
    map.set_chord("edit.redo", "Ctrl+R").unwrap();
    map.set_chord("draw.line", "Ctrl+Shift+Z").unwrap();
    assert_eq!(
        map.resolve_chord(&parse_chord("Ctrl+Shift+Z").unwrap()),
        Some("draw.line")
    );
    assert!(map.reset("edit.redo").is_err());
    assert_eq!(
        map.resolve_chord(&parse_chord("Ctrl+R").unwrap()),
        Some("edit.redo")
    );
}

#[test]
fn aliases_are_unique_and_resolve_without_case_sensitivity() {
    let mut map = sample_keymap();
    assert!(map.set_alias("draw.line", "L").is_err());
    map.set_alias("draw.line", "Line").unwrap();
    assert_eq!(map.resolve_alias("LINE"), Some("draw.line"));
    assert!(map.set_alias("edit.undo", "line").is_err());
    map.set_alias("draw.line", "").unwrap();
    assert_eq!(map.resolve_alias("line"), None);
}
#[test]
fn sketch_ready_alias_availability_matches_supported_actions() {
    let map = Keymap::desktop_defaults();
    assert!(map.supports_sketch_alias("sketch-rectangle"));
    assert!(map.supports_sketch_alias("edit.undo"));
    assert!(!map.supports_sketch_alias("view.pan"));
    assert!(!map.supports_sketch_alias("missing"));
}

#[test]
fn resetting_all_shortcuts_preserves_aliases() {
    let mut map = sample_keymap();
    map.set_alias("draw.line", "Line").unwrap();
    map.set_chord("edit.undo", "Ctrl+U").unwrap();
    map.reset_all_chords();
    assert_eq!(map.resolve_alias("line"), Some("draw.line"));
    assert_eq!(
        map.resolve_chord(&parse_chord("Ctrl+Z").unwrap()),
        Some("edit.undo")
    );
}
#[test]
fn navigation_keys_are_protected_from_other_shortcuts() {
    let mut map = Keymap::desktop_defaults();
    for chord in ["F2", "F3", "F4"] {
        assert!(map.set_chord("sketch-rectangle", chord).is_err());
    }
    map.set_chord("view.pan", "Ctrl+P").unwrap();
    map.set_chord("sketch-rectangle", "F2").unwrap();
}

#[test]
fn import_is_atomic_and_supports_swapping_chords() {
    let mut map = sample_keymap();
    let swapped = KeymapExport {
        chords: Some(BTreeMap::from([
            ("edit.undo".into(), "Ctrl+Y".into()),
            ("edit.redo".into(), "Ctrl+Z".into()),
        ])),
        aliases: None,
    };
    map.import(&swapped).unwrap();
    assert_eq!(
        map.resolve_chord(&parse_chord("Ctrl+Y").unwrap()),
        Some("edit.undo")
    );
    let invalid = KeymapExport {
        chords: Some(BTreeMap::from([("missing".into(), "Ctrl+Q".into())])),
        aliases: None,
    };
    assert!(map.import(&invalid).is_err());
    assert_eq!(map.export().chords, swapped.chords);
    map.reset_all();
    assert_eq!(
        map.resolve_chord(&parse_chord("Ctrl+Z").unwrap()),
        Some("edit.undo")
    );
}

#[test]
fn catalog_rejects_duplicate_ids_and_default_conflicts() {
    let one = KeymapAction::builtin("edit.undo", "Undo", "Ctrl+Z", &[]).unwrap();
    assert!(Keymap::new(vec![one.clone(), one]).is_err());
    let other = KeymapAction::command("draw.line", "Line", "Ctrl+Z").unwrap();
    assert!(
        Keymap::new(vec![
            KeymapAction::builtin("edit.undo", "Undo", "Ctrl+Z", &[]).unwrap(),
            other
        ])
        .is_err()
    );
}

#[test]
fn saved_unknown_bindings_survive_known_edits_without_becoming_actions() {
    let mut map = sample_keymap();
    let saved = KeymapExport {
        chords: Some(BTreeMap::from([("future.tool".into(), "Ctrl+Q".into())])),
        aliases: Some(BTreeMap::from([("future.tool".into(), "Future".into())])),
    };
    map.restore_saved(&saved).unwrap();
    assert_eq!(map.resolve_chord(&parse_chord("Ctrl+Q").unwrap()), None);
    assert_eq!(map.resolve_alias("Future"), None);
    assert!(map.set_alias("draw.line", "future").is_err());
    map.set_chord("edit.undo", "Ctrl+U").unwrap();
    let exported = map.export();
    assert_eq!(
        exported.chords.unwrap().get("future.tool").unwrap(),
        "Ctrl+Q"
    );
    assert_eq!(exported.aliases, saved.aliases);
}

#[test]
fn extrusion_shortcut_respects_remapping_and_collisions() {
    let mut map = Keymap::desktop_defaults();
    assert_eq!(
        map.resolve_chord(&parse_chord("E").unwrap()),
        Some("Create.Extrude")
    );
    map.set_chord("Create.Extrude", "Ctrl+E").unwrap();
    assert!(map.set_chord("sketch-line", "Ctrl+E").is_err());
    assert_eq!(map.resolve_chord(&parse_chord("E").unwrap()), None);
    assert_eq!(
        map.resolve_chord(&parse_chord("Ctrl+E").unwrap()),
        Some("Create.Extrude")
    );
    map.reset("Create.Extrude").unwrap();
    assert_eq!(
        map.resolve_chord(&parse_chord("E").unwrap()),
        Some("Create.Extrude")
    );
}

#[test]
fn saved_single_letter_assignment_keeps_ownership_over_a_new_default() {
    let mut map = Keymap::desktop_defaults();
    let saved = KeymapExport {
        chords: Some(BTreeMap::from([
            ("sketch-line".into(), "E".into()),
            ("sketch-circle".into(), "Shift+E".into()),
        ])),
        aliases: None,
    };
    map.restore_saved(&saved).unwrap();
    assert_eq!(
        map.resolve_chord(&parse_chord("E").unwrap()),
        Some("sketch-line")
    );
    assert_eq!(
        map.resolve_chord(&parse_chord("Shift+E").unwrap()),
        Some("sketch-circle")
    );
    assert_eq!(
        map.catalog()
            .iter()
            .find(|entry| entry.action_id == "Create.Extrude")
            .unwrap()
            .chord,
        ""
    );
    assert_eq!(map.export().chords.unwrap()["sketch-line"], "E");
}
