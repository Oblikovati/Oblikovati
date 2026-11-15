// SPDX-License-Identifier: GPL-2.0-only
//! Numeric input without an axis editor belongs to the command alias buffer.
use super::super::super::ApplicationWorld;
use super::{CENTER, choose_ellipse};
use cucumber::{given, then, when};
use oblikovati_model::sketch::{SketchPointInput, SketchPrimitiveCreation};

#[given("the fixture sketch is Ready with the retained circle and midpoint line")]
#[given("an editable sketch with existing geometry is Ready")]
fn retained_fixture(world: &mut ApplicationWorld) {
    world.start_empty_sketch();
    world.create_midpoint_line();
    create_retained_circle(world);
    world.pump(Vec::new());
    let line = world.line_point(0, 0.3);
    let circle = world.sketch_pixel_of([3.5, -3.]);
    world.remember_edited_sketch(vec![line, circle]);
    world.expect_ready();
}
fn create_retained_circle(world: &mut ApplicationWorld) {
    let edit = world.app().owner.active_sketch_edit().unwrap();
    world
        .app()
        .owner
        .document_commands(edit.document)
        .create_sketch_primitive(
            edit.sketch,
            &SketchPrimitiveCreation::Circle {
                center: SketchPointInput::New([3., -3.]),
                radius: 0.5,
                construction: false,
            },
        )
        .unwrap();
}
#[when("Ellipse is selected and a free center is clicked")]
fn center_selected(world: &mut ApplicationWorld) {
    choose_ellipse(world);
    world.sketch_click(CENTER);
}
#[then("Select first axis point appears without a visible numeric field")]
fn no_axis_editor(world: &mut ApplicationWorld) {
    world.expect_status("Select first axis point");
    assert!(
        world.state().sketch_hud.is_none(),
        "expected no axis numeric HUD"
    );
    assert!(
        world.window_text("##sketch-hud").is_empty(),
        "expected no rendered axis numeric HUD"
    );
}
#[when("the 8 key is pressed")]
fn type_eight(world: &mut ApplicationWorld) {
    world.type_text("8");
}
#[when("I press 6")]
fn type_six(world: &mut ApplicationWorld) {
    world.type_text("6");
}
#[then("the status shows Command alias input: 8")]
fn alias_eight(world: &mut ApplicationWorld) {
    world.expect_status("Command alias input: 8");
    world.save_capture("ellipse-12-alias-status");
    assert!(world.state().takes_command_alias);
    world.expect_edited_sketch_retained();
}
#[then("Command alias input displays 6 instead of an axis-length editor")]
fn alias_six(world: &mut ApplicationWorld) {
    world.expect_status("Command alias input: 6");
    world.save_capture("ellipse-20-alias-status");
    assert!(world.state().takes_command_alias);
    world.expect_edited_sketch_retained();
}
#[when("Escape is pressed")]
fn escape_alias(world: &mut ApplicationWorld) {
    world.escape();
}
#[then("Ready appears with no added ellipse visible")]
fn retained_after_alias(world: &mut ApplicationWorld) {
    world.expect_ready();
    world.expect_no_text("Command alias input:");
    world.expect_edited_sketch_retained();
}
