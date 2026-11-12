// SPDX-License-Identifier: GPL-2.0-only
//! Native interaction with accepted and rejected sketch dimension expressions.
use super::super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_api::{client::Client, wire::ParameterRenameArgs};
use oblikovati_viewport::ViewportPhysicalKey;

#[given("rectangle height d50 evaluates to 10.000 mm")]
fn ten_millimetre_height(world: &mut ApplicationWorld) {
    super::rectangle_two_point::typed_rectangle(world, "10", "6");
    let parameter = world
        .state()
        .parameters
        .into_iter()
        .find(|row| row.value.starts_with("10 "))
        .expect("expected the 10 mm dimension parameter");
    Client::new(world.app().owner.document_host())
        .parameters()
        .rename(&ParameterRenameArgs {
            name: parameter.name,
            new_name: "d50".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_annotation("10");
}
#[when("I edit d50 and enter 2 followed by multiplication")]
fn enter_incomplete(world: &mut ApplicationWorld) {
    world.open_dimension_label("10");
    world.expect_text("d50");
    world.type_text("2*");
    world.pump(Vec::new());
}
#[then("the incomplete expression 2* is red")]
fn incomplete_red(world: &mut ApplicationWorld) {
    let [red, green, blue] = world.run_in_window("Edit Dimension", "2*").color;
    assert!(red > 200 && green < 100 && blue < 100);
}
#[then("the message says The expression cannot be evaluated.")]
fn evaluation_warning(world: &mut ApplicationWorld) {
    world.expect_text("The expression cannot be evaluated.");
}
#[then("the editor remains open with the invalid expression selected")]
fn invalid_selected(world: &mut ApplicationWorld) {
    assert!(world.text_is_selected("2*"));
}
#[when("I press End and type 6")]
fn complete_expression(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::End, false);
    world.type_text("6");
    world.pump(Vec::new());
}
#[then("the completed expression 2*6 is black")]
fn complete_normal(world: &mut ApplicationWorld) {
    let [_, green, blue] = world.run_in_window("Edit Dimension", "2*6").color;
    assert!(green > 100 && blue > 100);
}
#[then("the editor closes and the reframed height annotation reads 12")]
fn accepted_twelve(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    world.expect_annotation("12");
}

#[given("the accepted d50 expression is 2 mm * 6 ul")]
fn accepted_arithmetic(world: &mut ApplicationWorld) {
    ten_millimetre_height(world);
    world.open_dimension_label("10");
    world.type_text("2*6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("12");
}
#[when("I replace it with d50 using individual d, 5 and 0 keys")]
fn enter_self_reference(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    assert!(world.state().dimension_tool_active);
    world.click(world.exact_run_center("12"));
    for letter in ["d", "5", "0"] {
        world.type_text(letter);
    }
    world.pump(Vec::new());
}
#[then("the completed reference appears black")]
fn self_reference_normal(world: &mut ApplicationWorld) {
    let [_, green, blue] = world.run_in_window("Edit Dimension", "d50").color;
    assert!(green > 100 && blue > 100);
}
#[then("Edit Dimension Value reports Failed to edit dimension value")]
fn self_reference_rejected(world: &mut ApplicationWorld) {
    world.expect_text("Failed to edit dimension value.");
}
#[when("I cancel the failed edit and reopen d50")]
fn cancel_and_reopen(world: &mut ApplicationWorld) {
    world.click_text("OK");
    world.escape();
    world.click(world.exact_run_center("12"));
}
#[then("the previous arithmetic expression is retained")]
fn previous_expression(world: &mut ApplicationWorld) {
    assert!(world.window_text("Edit Dimension").contains("2*6"));
}
#[then("the height annotation remains 12")]
fn height_remains(world: &mut ApplicationWorld) {
    world.expect_annotation("12");
}
#[when("I press Escape twice")]
fn escape_twice(world: &mut ApplicationWorld) {
    world.escape();
    world.escape();
}
#[then("the editor closes and General Dimension exits")]
fn dimension_exits(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    assert!(!world.state().dimension_tool_active);
}
