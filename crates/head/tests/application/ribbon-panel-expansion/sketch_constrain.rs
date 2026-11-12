// SPDX-License-Identifier: GPL-2.0-only
//! Sketch Constrain expansion while the part is outside sketch edit.
use super::super::workspace_layout::browser_feature::{
    SKETCH, expect_solid_visible, open_visible_extruded_part, right_click_row,
};
use super::{ApplicationWorld, click_expansion_arrow, diagnostics};
use cucumber::{given, then, when};

const WINDOW: &str = "##ribbon-additional-Constrain";

#[given("a part is active in the MCAD Software outside sketch edit")]
fn active_part(world: &mut ApplicationWorld) {
    world.add_part();
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
}
#[given("the full Sketch ribbon is selected")]
fn sketch_tab(world: &mut ApplicationWorld) {
    click_sketch_tab(world);
    world.expect_text("Constrain");
    for label in ["Line", "Finish Sketch", "Precise Input"] {
        assert!(
            !world.run_is_enabled(label),
            "{label}: expected unavailable outside sketch edit"
        );
    }
}
#[given("the Constrain additional-command area is closed")]
fn constrain_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[when("the user opens the Constrain additional-command area")]
fn open_constrain(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Constrain");
}
#[then("the additional area appears below the Sketch ribbon")]
fn area_below_ribbon(_: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry(WINDOW).expect("expected Constrain expansion");
    let (ribbon, size) = diagnostics::window_geometry("##application-ribbon").unwrap();
    assert!(position[1] >= ribbon[1] + size[1]);
}
#[then("its visible commands are unavailable outside sketch edit")]
fn commands_unavailable(world: &mut ApplicationWorld) {
    for label in ["Edit Coordinate System", "Constraint Inference Scope"] {
        world.expect_text(label);
        assert!(
            !world.run_is_enabled(label),
            "{label}: expected unavailable"
        );
    }
}
#[then("the Sketch additional area closes")]
fn area_closes(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(WINDOW).is_none());
}
#[then("the 3D Model ribbon is visible")]
fn model_visible(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
}
#[when("the user returns to the Sketch ribbon tab")]
fn return_sketch(world: &mut ApplicationWorld) {
    click_sketch_tab(world);
}
fn click_sketch_tab(world: &mut ApplicationWorld) {
    let center = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Sketch")
        .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .expect("expected the Sketch tab")
        .center();
    world.click(diagnostics::pointer_pixels(center));
}
#[then("the Sketch ribbon is visible with its Constrain additional area closed")]
fn sketch_area_stays_closed(world: &mut ApplicationWorld) {
    world.expect_text("Constrain");
    area_closes(world);
}
#[given("a part contains an existing editable sketch")]
fn part_with_sketch(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
}
#[given("the full Sketch ribbon is visible outside sketch edit")]
fn sketch_visible_outside_edit(world: &mut ApplicationWorld) {
    sketch_tab(world);
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
}
#[when("the user enters sketch edit from the model browser")]
fn edit_sketch(world: &mut ApplicationWorld) {
    right_click_row(world, SKETCH);
    world.click_text("Edit Sketch");
    world.pump(Vec::new());
}
#[then("sketch commands become available and the Exit group shows Finish")]
fn sketch_commands_available(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    assert!(world.run_is_enabled("Finish Sketch"));
    assert!(world.run_is_enabled("Line"));
}
#[then("Edit Coordinate System and Constraint Inference Scope are visible")]
fn constrain_commands_visible(world: &mut ApplicationWorld) {
    world.expect_text("Edit Coordinate System");
    world.expect_text("Constraint Inference Scope");
}
#[when("the user selects Finish without changing sketch geometry")]
fn finish_sketch(world: &mut ApplicationWorld) {
    world.finish_sketch();
    world.pump(Vec::new());
}
#[then("sketch edit ends and the part view returns")]
fn returns_to_part(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Part
    );
    world.expect_text("Start 2D Sketch");
}
#[then("the Constrain additional-command area closes")]
fn constrain_area_closes(world: &mut ApplicationWorld) {
    area_closes(world);
}
#[then("the original part geometry remains visible")]
fn original_part_visible(world: &mut ApplicationWorld) {
    expect_solid_visible(world);
}

#[given("a part containing an existing sketch is active in the MCAD Software")]
fn existing_sketch_part(world: &mut ApplicationWorld) {
    open_visible_extruded_part(world);
}
#[given("the full Sketch ribbon is selected outside sketch edit")]
fn select_sketch_outside_edit(world: &mut ApplicationWorld) {
    sketch_visible_outside_edit(world);
}
#[given("the Create additional-command area is closed")]
fn create_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("##ribbon-additional-Create").is_none());
}
#[when("the user opens the Create additional-command area")]
fn open_create(world: &mut ApplicationWorld) {
    click_expansion_arrow(world, "Create");
}
#[then("Create Block and Precise Input are visible but unavailable")]
fn create_commands_unavailable(world: &mut ApplicationWorld) {
    let window = "##ribbon-additional-Create";
    for label in ["CreateBlock", "PreciseInput"] {
        let run = world.run_in_window(window, label);
        assert!(run.alpha < u8::MAX, "{label}: expected unavailable");
    }
}
#[then("no sketch edit is active")]
fn sketch_edit_inactive(world: &mut ApplicationWorld) {
    assert!(world.app().owner.active_sketch_edit().is_none());
}

#[test]
fn create_expansion_opens_the_existing_precise_input_for_a_line() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.select_line_tool();
    assert!(world.state().precise_target.is_some());
    open_create(&mut world);
    let control = world.run_in_window("##ribbon-additional-Create", "PreciseInput");
    assert_eq!(control.alpha, u8::MAX);
    let center = diagnostics::pointer_pixels(control.center());
    world.click(center);
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Precise Input").is_some());
    let footer = world.run_in_window("##ribbon-additional-Create", "Create");
    world.click(diagnostics::pointer_pixels(footer.center()));
    let control = world.run_in_window("##application-ribbon", "PreciseInput");
    world.click(diagnostics::pointer_pixels(control.center()));
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Precise Input").is_none());
    world.close();
}
