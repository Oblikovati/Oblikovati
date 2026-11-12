// SPDX-License-Identifier: GPL-2.0-only
//! Native interaction with accepted and rejected sketch dimension expressions.
use super::super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};
use oblikovati_api::{
    client::Client,
    wire::{DriveDimensionArgs, ParameterRenameArgs, SketchArgs},
};
use oblikovati_viewport::ViewportPhysicalKey;
#[path = "line_fixture.rs"]
mod line_fixture;

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

#[given("d64 is 50 mm and source d63 is 20 mm")]
fn component_dimensions(world: &mut ApplicationWorld) {
    line_fixture::create(world);
    world.expect_annotation("20");
    world.expect_annotation("50");
    world.observed_view = Some(world.line_point(0, 1.));
}
#[when("I edit d64 using individual d, 6, 3 and multiplication keys")]
fn enter_dependent_prefix(world: &mut ApplicationWorld) {
    world.open_dimension_label("50");
    world.expect_text("d64");
    for key in ["d", "6", "3", "*"] {
        world.type_text(key);
    }
    world.pump(Vec::new());
}
#[then("the incomplete multiplication expression is red")]
fn dependent_prefix_red(world: &mut ApplicationWorld) {
    let [red, green, blue] = world.run_in_window("Edit Dimension", "d63*").color;
    assert!(red > 200 && green < 100 && blue < 100);
}
#[when("I type 2")]
fn finish_dependent(world: &mut ApplicationWorld) {
    world.type_text("2");
    world.pump(Vec::new());
}
#[then("the completed expression is black")]
fn dependent_normal(world: &mut ApplicationWorld) {
    let [_, green, blue] = world.run_in_window("Edit Dimension", "d63*2").color;
    assert!(green > 100 && blue > 100);
}
#[then("its annotation displays fx:40")]
fn dependent_forty(world: &mut ApplicationWorld) {
    world.expect_annotation("fx:40");
}
#[then("the line visibly changes and its driven aligned length displays (44.721)")]
fn aligned_changes(world: &mut ApplicationWorld) {
    assert_ne!(world.line_point(0, 1.), world.observed_view.unwrap());
    world.expect_annotation("(44.721)");
}

#[test]
fn native_dimension_reference_updates_annotation_and_geometry_through_history() {
    let mut world = ApplicationWorld::new();
    ten_millimetre_height(&mut world);
    let parameters = world.state().parameters;
    let source = parameters
        .into_iter()
        .find(|row| row.name != "d50")
        .expect("expected the 6 mm source dimension");
    Client::new(world.app().owner.document_host())
        .parameters()
        .rename(&ParameterRenameArgs {
            name: source.name,
            new_name: "d63".into(),
        })
        .unwrap();
    world.pump(Vec::new());
    world.open_dimension_label("10");
    world.type_text("d63*2");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:12");
    world.open_dimension_label("6");
    world.type_text("8");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:16");
    world.undo();
    world.expect_annotation("fx:12");
    world.redo();
    world.expect_annotation("fx:16");
    world.open_dimension_label("fx:16");
    assert!(world.window_text("Edit Dimension").contains("d63*2"));
    world.close();
}

#[test]
fn native_reference_dimension_shows_parentheses_and_cannot_be_edited() {
    let mut world = ApplicationWorld::new();
    ten_millimetre_height(&mut world);
    let client = Client::new(world.app().owner.document_host());
    let dimensions = client.sketch().dimensions(&SketchArgs::default()).unwrap();
    let index = dimensions
        .dimensions
        .unwrap()
        .into_iter()
        .find(|dimension| dimension.name == "d50")
        .unwrap()
        .index;
    client
        .dimension()
        .edit(&DriveDimensionArgs {
            dimension_index: index,
            set_driven: true,
            driven: true,
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
    world.expect_annotation("(10)");
    world.open_dimension_label("(10)");
    assert!(diagnostics::window_geometry("Edit Dimension").is_none());
    world.close();
}
