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
#[given("d50 has accepted the arithmetic expression 2*6")]
#[given("d50 retains 2 mm * 6 ul and evaluates to 12 mm")]
fn accepted_arithmetic(world: &mut ApplicationWorld) {
    ten_millimetre_height(world);
    world.open_dimension_label("10");
    world.type_text("2*6");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("12");
    world.observed_counts = vec![rectangle_height(world)];
}
fn rectangle_height(world: &mut ApplicationWorld) -> usize {
    let points: Vec<_> = (0..4)
        .flat_map(|line| [world.line_point(line, 0.)[1], world.line_point(line, 1.)[1]])
        .collect();
    (points.iter().max().unwrap() - points.iter().min().unwrap()) as usize
}
#[when("I invoke General Dimension with d and single-click the existing annotation")]
#[when("I invoke General Dimension and single-click d50")]
fn open_d50_from_dimension_tool(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    assert!(world.state().dimension_tool_active);
    world.click(world.exact_run_center("12"));
}
#[then("Edit Dimension d50 shows 2 mm * 6 ul")]
#[then("the retained expression is 2 mm * 6 ul")]
fn normalized_arithmetic(world: &mut ApplicationWorld) {
    world.expect_text("d50");
    assert!(world.window_text("Edit Dimension").contains("2mm*6ul"));
}
#[then("the unchanged editor closes and General Dimension exits")]
fn unchanged_editor_exits(world: &mut ApplicationWorld) {
    dimension_exits(world);
}
#[then("the annotation reads 10 and the rectangle is shorter")]
fn undo_shorter(world: &mut ApplicationWorld) {
    world.expect_annotation("10");
    assert!(rectangle_height(world) < world.observed_counts[0]);
}
#[then("the annotation reads 12 and the rectangle is taller")]
fn redo_taller(world: &mut ApplicationWorld) {
    world.expect_annotation("12");
    assert_eq!(rectangle_height(world), world.observed_counts[0]);
}
#[when("I replace its selected expression with 9")]
fn propose_nine(world: &mut ApplicationWorld) {
    world.type_text("9");
    world.pump(Vec::new());
}
#[then("the uncommitted value is black")]
fn proposal_normal(world: &mut ApplicationWorld) {
    let [_, green, blue] = world.run_in_window("Edit Dimension", "9").color;
    assert!(green > 100 && blue > 100);
}
#[when("I press Escape and reopen d50")]
fn discard_and_reopen(world: &mut ApplicationWorld) {
    world.escape();
    world.click(world.exact_run_center("12"));
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
    assert!(world.window_text("Edit Dimension").contains("2mm*6ul"));
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
#[given("d64 has accepted d63*2")]
fn dependent_line(world: &mut ApplicationWorld) {
    component_dimensions(world);
    world.open_dimension_label("50");
    world.type_text("d63*2");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:40");
}
#[when("I open source d63 and type 3 then 0 and press Return")]
fn change_source(world: &mut ApplicationWorld) {
    world.open_dimension_label("20");
    world.type_text("3");
    world.type_text("0");
    world.key(ViewportPhysicalKey::Enter, false);
}
#[then("the source annotation displays 30 and the dependent annotation displays fx:60")]
fn propagated_annotations(world: &mut ApplicationWorld) {
    world.expect_annotation("30");
    world.expect_annotation("fx:60");
}
#[then("the driven aligned length displays (67.082)")]
fn aligned_length_sixty(world: &mut ApplicationWorld) {
    world.expect_annotation("(67.082)");
}
#[when("I reopen d64 by targeting its component dimension line")]
fn reopen_by_line(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::D, false);
    let point = world.sketch_pixel_of([5., 2.5]);
    world.sketch_click(point);
    world.expect_text("d64");
}
#[then("the retained normalized expression is d63 * 2 ul")]
fn normalized_dependent(world: &mut ApplicationWorld) {
    assert!(world.window_text("Edit Dimension").contains("d63*2ul"));
}

#[given("d64 depends on d63*2 and source d63 has been changed to 30")]
#[given("d63 is 30 and d64 retains d63 * 2 ul")]
fn propagated_line(world: &mut ApplicationWorld) {
    component_dimensions(world);
    world.open_dimension_label("50");
    world.type_text("d63*2");
    world.key(ViewportPhysicalKey::Enter, false);
    world.open_dimension_label("20");
    world.type_text("30");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:60");
}
#[when("I open d63 and type d,6,4 as separate keys and press Return")]
fn indirect_reference(world: &mut ApplicationWorld) {
    world.open_dimension_label("30");
    for character in ["d", "6", "4"] {
        world.type_text(character);
    }
    world.key(ViewportPhysicalKey::Enter, false);
}
#[when("I click Cancel and reopen d63")]
fn cancel_indirect_reference(world: &mut ApplicationWorld) {
    world.click_text("Cancel");
    world.open_dimension_label("30");
}
#[then("the selected retained value is 30 mm")]
fn source_value_retained(world: &mut ApplicationWorld) {
    assert!(world.text_is_selected("30mm"));
}
#[then("dependent fx:60 and driven length (67.082) remain visible")]
fn dependent_geometry_retained(world: &mut ApplicationWorld) {
    world.expect_annotation("fx:60");
    world.expect_annotation("(67.082)");
}
#[when("I select both line endpoints in Measure")]
fn measure_endpoints(world: &mut ApplicationWorld) {
    world.key(ViewportPhysicalKey::M, false);
    for fraction in [0., 1.] {
        let point = world.line_point(0, fraction);
        world.sketch_click(point);
    }
}
#[then("both selections are Points")]
fn points_selected(world: &mut ApplicationWorld) {
    world.expect_measure("Selection 1: Point");
    world.expect_measure("Selection 2: Point");
}
fn expect_measurement(world: &mut ApplicationWorld, y: &str, z: &str, distance: &str) {
    world.expect_measure(&format!("Minimum distance {distance} mm"));
    world.expect_measure(&format!("Minimum delta X 0.000 Y {y} Z {z} mm"));
}
#[then("Delta Y is 30.000 mm and Delta Z is 60.000 mm and distance is 67.082 mm")]
#[then("the 30/60 deltas and 67.082 mm distance return")]
fn measure_thirty_sixty(world: &mut ApplicationWorld) {
    expect_measurement(world, "30.000", "60.000", "67.082");
}
#[when("I close Measure, undo the source edit and measure both endpoints again")]
fn measure_after_undo(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.undo();
    measure_endpoints(world);
}
#[then("Delta Y is 20.000 mm and Delta Z is 40.000 mm and distance is 44.721 mm")]
fn measure_twenty_forty(world: &mut ApplicationWorld) {
    expect_measurement(world, "20.000", "40.000", "44.721");
}
#[when("I close Measure, redo the source edit and measure both endpoints again")]
fn measure_after_redo(world: &mut ApplicationWorld) {
    world.click_text("Done");
    world.redo();
    measure_endpoints(world);
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
    assert!(world.window_text("Edit Dimension").contains("d63*2ul"));
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:16");
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

#[test]
fn native_dimension_dependency_survives_save_close_and_reopen() {
    let mut world = ApplicationWorld::new();
    propagated_line(&mut world);
    world.finish_sketch();
    world.press_save_shortcut();
    let path = world.accept_save_as();
    world.close_active_tab();
    world.reopen_through_dialog(&path);
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text.starts_with("[DOF"))
        .expect("expected the saved sketch in the browser")
        .center()
        .map(|coordinate| coordinate as i32);
    world.double_click(row);
    world.pump(Vec::new());
    world.expect_annotation("fx:60");
    world.open_dimension_label("fx:60");
    assert!(world.window_text("Edit Dimension").contains("d63*2ul"));
    world.escape();
    world.open_dimension_label("30");
    world.type_text("40");
    world.key(ViewportPhysicalKey::Enter, false);
    world.expect_annotation("fx:80");
    world.expect_annotation("(89.443)");
    world.close();
}
