// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use selector_keyboard::model_panel_run;

fn ribbon_panel_x(world: &ApplicationWorld, panel: &str) -> i32 {
    model_panel_run(world, panel)
        .unwrap_or_else(|| panic!("expected {panel:?} in the ribbon"))
        .center()[0] as i32
}

fn modify_positions(world: &ApplicationWorld) -> [i32; 2] {
    let (position, _) = super::super::super::diagnostics::window_geometry(EXPANSION)
        .expect("expected the Modify additional commands");
    [ribbon_panel_x(world, "Modify"), position[0] as i32]
}

#[given("the Create and Modify panels are visible, with Create before Modify")]
fn create_before_modify(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
    assert!(ribbon_panel_x(world, "Create") < ribbon_panel_x(world, "Modify"));
}

#[when("the user opens the panel selector and clears Create")]
fn hide_create(world: &mut ApplicationWorld) {
    world.observed_view = Some(modify_positions(world));
    side_selector::open_side_selector(world);
    assert!(world.checked("Create"));
    world.click_text("Create");
}

#[then("Create is unchecked and its panel is hidden")]
fn create_hidden(world: &mut ApplicationWorld) {
    assert!(!world.checked("Create"));
    world.expect_no_text("Extrude");
}

#[then("the Modify panel shifts left while its additional commands remain open")]
fn modify_shifts_left(world: &mut ApplicationWorld) {
    let [main, expansion] = modify_positions(world);
    let [before_main, before_expansion] = world.observed_view.expect("expected prior positions");
    assert!(main < before_main && expansion < before_expansion);
    world.run_in_window(EXPANSION, "MoveBodies");
}

#[when("the user selects Create in the panel selector")]
fn show_create(world: &mut ApplicationWorld) {
    assert!(side_selector::selector_popup_open());
    world.click_text("Create");
}

#[then("Create is checked and returns before Modify")]
fn create_restored(world: &mut ApplicationWorld) {
    assert!(world.checked("Create"));
    create_before_modify(world);
}

#[then("the Modify panel and its open additional commands return to their earlier position")]
#[then("Modify and its open additional command area return to their earlier position")]
fn modify_positions_restored(world: &mut ApplicationWorld) {
    assert_eq!(Some(modify_positions(world)), world.observed_view);
    world.run_in_window(EXPANSION, "MoveBodies");
}

#[given("Primitives is hidden and Create and Modify are visible")]
fn primitives_before_create_hidden(world: &mut ApplicationWorld) {
    assert!(model_panel_run(world, "Primitives").is_none());
    create_before_modify(world);
}

#[when("the user opens the panel selector and selects Primitives")]
fn show_primitives(world: &mut ApplicationWorld) {
    world.observed_view = Some(modify_positions(world));
    side_selector::open_side_selector(world);
    assert!(!world.checked("Primitives"));
    click_primitives_checkbox(world);
}

#[when("the user clears Primitives in the panel selector")]
fn hide_primitives(world: &mut ApplicationWorld) {
    assert!(world.checked("Primitives"));
    click_primitives_checkbox(world);
}

fn click_primitives_checkbox(world: &mut ApplicationWorld) {
    assert!(side_selector::selector_popup_open());
    let panel_list = model_panel_run(world, "Start 2D Sketch").unwrap().list;
    let row = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == "Primitives" && run.list != panel_list)
        .expect("expected the Primitives checkbox in the panel selector");
    world.click(super::super::super::diagnostics::pointer_pixels(
        row.center(),
    ));
}

#[then("Primitives is checked and appears before Create")]
fn primitives_precedes_create(world: &mut ApplicationWorld) {
    assert!(world.checked("Primitives"));
    assert!(ribbon_panel_x(world, "Primitives") < ribbon_panel_x(world, "Create"));
}

#[then("Modify and its open additional command area shift to the right")]
fn modify_shifts_right(world: &mut ApplicationWorld) {
    let [main, expansion] = modify_positions(world);
    let [before_main, before_expansion] = world.observed_view.expect("expected prior positions");
    let [panel_shift, expansion_shift] = [main - before_main, expansion - before_expansion];
    assert!(
        panel_shift > 0 && expansion_shift > 0 && panel_shift.abs_diff(expansion_shift) <= 1,
        "expected Modify and its expansion to move together, got {panel_shift} and {expansion_shift}"
    );
    world.run_in_window(EXPANSION, "MoveBodies");
    eprintln!(
        "Modify positions before = {before_main}, {before_expansion}; after = {main}, {expansion}"
    );
}
