// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use cucumber::{given, then, when};

#[given("the ribbon is docked on the right with the 3D Model tab selected")]
fn right_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    choose_position(world, "Right");
}
#[given("Modify and Work Features are displayed as compact group buttons")]
fn group_buttons(world: &mut ApplicationWorld) {
    world.expect_text("Modify");
    world.expect_text("Work Features");
    expect_vertical(world, &["Modify", "Work Features"]);
}
#[when("the user selects the Modify group button")]
fn select_modify(world: &mut ApplicationWorld) {
    world.click_text("Modify");
    world.pump(Vec::new());
}
#[when("the user selects the Work Features group button")]
fn select_work(world: &mut ApplicationWorld) {
    world.click_text("Work Features");
    world.pump(Vec::new());
}
fn expect_left_of_column(world: &ApplicationWorld, command: &str) {
    let (position, _) = window_geometry(COLUMN).expect("expected the right ribbon column");
    let command_x = world.run(command).center()[0];
    assert!(
        command_x < position[0],
        "{command} at {command_x}: expected left of {position:?}"
    );
}
#[then("a temporary Modify panel opens to the left of the ribbon")]
fn modify_left(world: &mut ApplicationWorld) {
    expect_left_of_column(world, "Fillet");
}
#[then("the panel displays Hole, Fillet and other Modify commands")]
fn modify_commands(world: &mut ApplicationWorld) {
    for command in ["Hole", "Fillet", "Chamfer"] {
        world.expect_text(command);
    }
}
fn expect_highlight(world: &mut ApplicationWorld, selected: &str, other: &str) {
    let points = [selected, other].map(|label| {
        let run = world.run(label);
        [run.minimum[0] as usize - 4, run.center()[1] as usize]
    });
    assert_ne!(
        world.pixel_at(points[0]),
        world.pixel_at(points[1]),
        "expected {selected} highlighted"
    );
}
#[then("the Modify group button is highlighted")]
fn modify_highlighted(world: &mut ApplicationWorld) {
    expect_highlight(world, "Modify", "Work Features");
}
#[then("a temporary Work Features panel opens to the left of the ribbon")]
fn work_left(world: &mut ApplicationWorld) {
    expect_left_of_column(world, "UCS");
}
#[then("the panel displays Plane, Axis, Point and UCS")]
fn work_commands(world: &mut ApplicationWorld) {
    for command in ["Plane", "Axis", "Point", "UCS"] {
        world.expect_text(command);
    }
}
#[then("the Work Features group button is highlighted")]
fn work_highlighted(world: &mut ApplicationWorld) {
    expect_highlight(world, "Work Features", "Modify");
}
#[when("the user docks the ribbon at the top")]
fn dock_top(world: &mut ApplicationWorld) {
    choose_position(world, "Top");
}
#[then("the temporary Work Features panel closes")]
fn work_closed(_world: &mut ApplicationWorld) {
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("the 3D Model ribbon is displayed above the viewport")]
fn model_top(world: &mut ApplicationWorld) {
    expect_top_dock(world);
    same_part(world);
}

#[then("the part remains visible")]
fn part_remains(world: &mut ApplicationWorld) {
    same_part(world);
    world.expect_text("Origin");
}
