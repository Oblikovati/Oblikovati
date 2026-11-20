// SPDX-License-Identifier: GPL-2.0-only
use super::super::x11_input;
use super::{ApplicationWorld, diagnostics, select_sketch_tab};
use crate::HeadSketchWork;
use cucumber::{given, then, when};
#[path = "text_bold.rs"]
mod text_bold;
#[path = "text_creation.rs"]
mod text_creation;
#[path = "text_double_click.rs"]
mod text_double_click;
#[path = "text_edit.rs"]
mod text_edit;
#[path = "text_format_choices.rs"]
mod text_format_choices;
#[path = "text_numbering.rs"]
mod text_numbering;
#[path = "text_presets.rs"]
mod text_presets;
use oblikovati_viewport::ViewportSketchPreview;

#[test]
fn sketch_text_ribbon_accepts_cancels_and_round_trips_in_history() {
    let mut world = ApplicationWorld::new();
    start_text_from_part(&mut world);
    open_editor(&mut world, [850, 550]);
    type_content(&mut world, b'a');
    click_editor_button(&mut world, "Cancel");
    assert!(world.state().sketch_text.is_none());
    assert!(world.app().owner.sketch_preview().is_none());
    assert!(world.edited_geometry().texts().is_empty());
    open_editor(&mut world, [900, 580]);
    type_content(&mut world, b'b');
    click_editor_button(&mut world, "OK");
    assert!(world.state().sketch_text.is_none());
    let text = world.edited_geometry().texts()[0].clone();
    assert_eq!(text.style.content, "b");
    assert_eq!(text.style.height, 0.305);
    assert!(
        world
            .app()
            .window
            .native
            .sketch_display_declines()
            .unwrap()
            .is_empty()
    );
    check_history(&mut world, &text);
    world.close();
}

fn start_text_from_part(world: &mut ApplicationWorld) {
    world.add_part();
    world.start_sketch();
    world.finish_sketch();
    activate_text(world);
}

fn activate_text(world: &mut ApplicationWorld) {
    select_sketch_tab(world);
    assert!(world.run_is_enabled("Text"));
    x11_input::focus_window();
    x11_input::click(diagnostics::pointer_pixels(world.run("Text").center()));
    world.pump(Vec::new());
    assert!(world.app().owner.active_sketch_edit().is_some());
    assert!(!world.state().sketch_at_ready);
}

#[given("the user canceled new text containing lowercase a without accepting it")]
fn cancel_pending_sample(world: &mut ApplicationWorld) {
    start_text_from_part(world);
    open_editor(world, [850, 550]);
    type_content(world, b'a');
    click_editor_button(world, "Cancel");
    assert!(world.state().sketch_text.is_none());
    assert!(world.edited_geometry().texts().is_empty());
}

#[when("the user starts Text again and choose a new insertion location")]
fn start_again_after_cancel(world: &mut ApplicationWorld) {
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    x11_input::click(diagnostics::pointer_pixels(world.run("Text").center()));
    world.pump(Vec::new());
    open_editor(world, [900, 580]);
}

#[then("Format Text opens with an empty content editor")]
fn new_content_is_empty(world: &mut ApplicationWorld) {
    world.expect_text("Format Text");
    assert!(world.state().sketch_text.unwrap().content.is_empty());
}

#[then("the user records the retained formatting defaults separately from content")]
fn observe_retained_text_format(world: &mut ApplicationWorld) {
    let draft = world.state().sketch_text.unwrap();
    assert_eq!(draft.height_mm, "3.05");
    assert_eq!(draft.rotation_degrees, "0");
    assert!(draft.family.is_empty());
    world.run_in_window("Format Text", "Height(mm)");
    if std::env::var_os("OBK_BDD_REPORTS").is_some() {
        world.save_capture("cancelled-text-fresh-editor");
    }
}

fn open_editor(world: &mut ApplicationWorld, anchor: [i32; 2]) {
    x11_input::click(anchor);
    world.pump(Vec::new());
    assert!(world.state().sketch_text.unwrap().content.is_empty());
    world.expect_text("Format Text");
}

fn type_content(world: &mut ApplicationWorld, letter: u8) {
    focus_content(world);
    x11_input::press_keysym(u32::from(letter));
    world.pump(Vec::new());
    let Some(ViewportSketchPreview::Text { style, .. }) = world.app().owner.sketch_preview() else {
        panic!("expected the text preview after typing");
    };
    assert_eq!(style.content, char::from(letter).to_string());
    assert!(world.edited_geometry().texts().is_empty());
}

fn click_editor_button(world: &mut ApplicationWorld, label: &str) {
    let position = world.run_in_window("Format Text", label).center();
    x11_input::click(diagnostics::pointer_pixels(position));
    world.pump(Vec::new());
}

fn check_history(world: &mut ApplicationWorld, text: &oblikovati_model::sketch::SketchText2D) {
    x11_input::press_keysym(0xff1b);
    world.pump(Vec::new());
    assert!(world.state().sketch_at_ready);
    x11_input::press_control_letter(b'z');
    world.pump(Vec::new());
    assert!(world.edited_geometry().texts().is_empty());
    x11_input::press_control_letter(b'y');
    world.pump(Vec::new());
    assert_eq!(world.edited_geometry().texts(), std::slice::from_ref(text));
}

fn focus_content(world: &mut ApplicationWorld) {
    let label = world.run_in_window("Format Text", "Content").clone();
    let x = label.minimum[0] - 80.;
    let y = label.minimum[1] + 8.;
    x11_input::click(diagnostics::pointer_pixels([x, y]));
    world.pump(Vec::new());
}
