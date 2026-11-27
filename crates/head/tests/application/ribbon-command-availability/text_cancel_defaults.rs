// SPDX-License-Identifier: GPL-2.0-only
//! Verify that cancelling a placement retains the configured formatting default.
use super::*;

pub(super) const TAHOMA: &str = "Tahoma";

#[test]
fn native_cancelled_text_uses_the_blank_formatting_default_on_restart() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let original = world.edited_geometry();
    cancel_draft_with_font_default(&mut world, "DejaVu Sans");
    restart_cancelled_text(&mut world);
    expect_blank_formatting_default(&mut world, "DejaVu Sans");
    assert_eq!(world.edited_geometry(), original);
    world.close();
}

#[given("the earlier new text containing a was canceled")]
fn cancel_recorded_tahoma_draft(world: &mut ApplicationWorld) {
    cancel_draft_with_font_default(world, TAHOMA);
}

#[when("the user starts Text and choose another insertion point")]
fn restart_cancelled_text(world: &mut ApplicationWorld) {
    start_again_after_cancel(world);
}

#[then("the font is Tahoma, height is 3.05 mm and width is 100")]
fn expect_tahoma_formatting_default(world: &mut ApplicationWorld) {
    expect_blank_formatting_default(world, TAHOMA);
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("text-after-cancel-with-tahoma-default");
    }
}

fn cancel_draft_with_font_default(world: &mut ApplicationWorld, family: &str) {
    configure_blank_font_default(world, family);
    open_editor(world, [850, 550]);
    expect_blank_formatting_default(world, family);
    type_content(world, b'a');
    text_format_choices::choose_format(world, "Choose height", "6.10 mm");
    click_editor_button(world, "Cancel");
    assert!(world.app().owner.sketch_preview().is_none());
}

/// Configure a content-free default through Format Text, for example with Tahoma.
pub(super) fn configure_blank_font_default(world: &mut ApplicationWorld, family: &str) {
    activate_text(world);
    open_editor(world, [850, 550]);
    text_format_choices::choose_font(world, family);
    // A content-free preset supplies the fixture's font without adding sketch geometry.
    text_presets::click_in_text_editor(world, "+");
    text_presets::confirm_preset_name(world);
    world.run_in_window("Format Text", "Preset");
    text_preset_defaults::set_new_text_default(world, "Use Current Preset");
}

/// Check the empty draft and its configured format, for example Tahoma at 3.05 mm.
pub(super) fn expect_blank_formatting_default(world: &mut ApplicationWorld, family: &str) {
    let draft = world.state().sketch_text.unwrap();
    assert!(draft.content.is_empty());
    assert_eq!(draft.family, family);
    assert_eq!(draft.height_mm, "3.05");
    assert!(draft.formatting.runs.is_none());
    let drawn_family: String = family.split_whitespace().collect();
    world.run_in_window("Format Text", &drawn_family);
    world.run_in_window("Format Text", "3.05");
    world.run_in_window("Format Text", "100");
    let Some(ViewportSketchPreview::Text {
        style, replaces, ..
    }) = world.app().owner.sketch_preview()
    else {
        panic!("expected an active empty Text placement");
    };
    assert!(style.content.is_empty());
    assert!(replaces.is_none());
    assert!(world.edited_geometry().texts().is_empty());
}
