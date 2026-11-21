// SPDX-License-Identifier: GPL-2.0-only
//! Exercise session preset state across drafts and placement modes.
use super::*;
#[test]
fn saved_preset_can_apply_to_a_new_draft() {
    let saved = GuiSketchTextDraft {
        content: "ab".into(),
        ..Default::default()
    };
    let mut shelf = TextPresets::default();
    assert!(shelf.save_name(&suggest_name(&saved.content), &saved));
    assert!(!shelf.save_name("AB", &saved));
    shelf.initial = shelf.current.clone();
    shelf.synchronize(true);
    let mut next = GuiSketchTextDraft::default();
    assert!(shelf.begin(false, None, &mut next));
    assert_eq!(next, saved);
}

#[test]
fn last_used_requires_a_submitted_draft_and_survives_later_cancel() {
    let draft = GuiSketchTextDraft {
        content: "ab".into(),
        ..Default::default()
    };
    let mut shelf = TextPresets::default();
    shelf.synchronize(true);
    shelf.submit(&draft);
    shelf.synchronize(false);
    shelf.synchronize(true);
    shelf.submit(&GuiSketchTextDraft::default());
    shelf.cancel();
    shelf.synchronize(false);
    assert_eq!(shelf.fields(&PresetChoice::LastUsed), Some(&draft));
}

#[test]
fn last_used_keeps_the_named_origin_until_the_preset_is_removed() {
    let saved = GuiSketchTextDraft {
        content: "ab".into(),
        ..Default::default()
    };
    let modified = GuiSketchTextDraft {
        content: "AB".into(),
        ..saved.clone()
    };
    let mut shelf = TextPresets::default();
    assert!(shelf.save_name("g", &saved));
    let origin = shelf.selected_preset();
    shelf.synchronize(true);
    shelf.submit(&modified);
    shelf.synchronize(false);
    shelf.initial = PresetChoice::LastUsed;
    shelf.synchronize(true);
    let mut draft = GuiSketchTextDraft::default();
    assert!(shelf.begin(false, None, &mut draft));
    assert_eq!(draft, modified);
    assert_eq!(shelf.caption(&draft), "*g");
    assert_eq!(shelf.selected_preset(), origin);
    assert_eq!(shelf.current_index(), None);
    shelf.entries.clear();
    assert_eq!(shelf.caption(&draft), LAST);
    assert_eq!(shelf.selected_preset(), None);
    assert_eq!(shelf.fields(&PresetChoice::LastUsed), Some(&modified));
}

#[test]
fn reopened_modified_text_keeps_its_preset_through_rename() {
    let mut shelf = TextPresets::default();
    let original = GuiSketchTextDraft {
        content: "ab".into(),
        ..Default::default()
    };
    assert!(shelf.save_name("f", &original));
    let preset = shelf.selected_preset();
    let mut changed = original.clone();
    changed.content = "c".into();
    shelf.synchronize(true);
    assert!(!shelf.begin(true, preset, &mut changed));
    assert_eq!(shelf.caption(&changed), "*f");
    shelf.name = Some(PresetName {
        value: "g".into(),
        rename: true,
        focus: false,
    });
    assert!(shelf.save_name("g", &changed));
    assert_eq!(shelf.selected_preset(), preset);
    assert_eq!(shelf.caption(&changed), "*g");
    assert_eq!(shelf.fields(&shelf.current), Some(&original));
}

#[test]
fn matching_content_does_not_assign_a_preset_to_unrelated_text() {
    let mut shelf = TextPresets::default();
    let mut draft = GuiSketchTextDraft {
        content: "ab".into(),
        ..Default::default()
    };
    assert!(shelf.save_name("f", &draft));
    shelf.synchronize(true);
    assert!(!shelf.begin(true, None, &mut draft));
    assert_eq!(shelf.caption(&draft), NONE);
}
