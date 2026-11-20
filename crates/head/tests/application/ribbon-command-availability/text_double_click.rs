// SPDX-License-Identifier: GPL-2.0-only
//! Open retained Sketch Text from its glyph after a native undo and redo.
use super::*;

#[test]
fn native_double_click_reopens_redone_bold_text_and_cancel_preserves_it() {
    let mut world = ApplicationWorld::new();
    create_and_restore_bold(&mut world, "DejaVu Sans");
    let at = text_edit::glyph_pixel(&mut world);
    x11_input::click(at);
    world.pump(Vec::new());
    assert!(world.state().sketch_text.is_none());
    x11_input::click([450, 680]);
    x11_input::click([450, 680]);
    world.pump(Vec::new());
    assert!(world.state().sketch_text.is_none());
    double_click_text(&mut world);
    expect_retained_format(&mut world, "DejaVu Sans");
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("reopened-bold-text");
    }
    cancel_reopened_text(&mut world);
    accepted_text_remains(&mut world);
    world.close();
}

#[given("the bold text creation was restored by Redo")]
fn restored_tahoma_text(world: &mut ApplicationWorld) {
    create_and_restore_bold(world, "Tahoma");
}

fn create_and_restore_bold(world: &mut ApplicationWorld, family: &str) {
    start_text_from_part(world);
    open_editor(world, [900, 580]);
    text_format_choices::choose_font(world, family);
    type_content(world, b'a');
    x11_input::press_keysym(u32::from(b'b'));
    world.pump(Vec::new());
    x11_input::press_control_a();
    world.pump(Vec::new());
    click_editor_button(world, "Bold");
    click_editor_button(world, "OK");
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    record_restored_text(world);
}

fn record_restored_text(world: &mut ApplicationWorld) {
    let accepted = world.edited_geometry();
    assert_eq!(accepted.texts()[0].style.content, "ab");
    let count = world.dimensions_needed();
    text_edit::undo_edit(world);
    assert!(world.edited_geometry().texts().is_empty());
    text_edit::redo_edit(world);
    assert_eq!(world.edited_geometry(), accepted);
    world.text_creation_baseline = Some(accepted);
    world.observed_counts = vec![count];
}

#[when("the user double-clicks the text in one supported gesture")]
fn double_click_text(world: &mut ApplicationWorld) {
    let at = text_edit::glyph_pixel(world);
    x11_input::click(at);
    x11_input::click(at);
    world.pump(Vec::new());
    world.expect_text("Format Text");
}

#[then("Format Text shows Bold highlighted, Tahoma, 3.05 mm and both entered characters")]
fn expect_tahoma_format(world: &mut ApplicationWorld) {
    expect_retained_format(world, "Tahoma");
}

fn expect_retained_format(world: &mut ApplicationWorld, family: &str) {
    let draft = world.state().sketch_text.unwrap();
    assert_eq!(draft.family, family);
    assert_eq!(draft.height_mm, "3.05");
    assert_eq!(draft.content, "ab");
    let [run] = draft.formatting.runs.as_deref().unwrap() else {
        panic!("expected one retained Bold run");
    };
    assert_eq!((run.start, run.end, run.bold), (0, 2, true));
    world.run_in_window("Format Text", "Bold");
    world.expect_text("ab");
}

#[when("the user cancels the unchanged editor")]
fn cancel_reopened_text(world: &mut ApplicationWorld) {
    click_editor_button(world, "Cancel");
}

#[then("the accepted text remains")]
fn accepted_text_remains(world: &mut ApplicationWorld) {
    world.expect_ready();
    assert_eq!(
        &world.edited_geometry(),
        world.text_creation_baseline.as_ref().unwrap()
    );
    assert_eq!(world.dimensions_needed(), world.observed_counts[0]);
}
