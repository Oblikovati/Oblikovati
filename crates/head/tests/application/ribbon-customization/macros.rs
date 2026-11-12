// SPDX-License-Identifier: GPL-2.0-only
//! Native scenario for the empty macro category and reusable separator.
use super::*;

#[given("the Ribbon customization dialog shows an empty Part Tools target")]
fn empty_target(world: &mut ApplicationWorld) {
    open_tools_customization(world);
    assert!(!world.window_text("Customize").contains("Part|Tools*"));
}

#[given("the Macros category has no macro commands available in this session")]
fn no_macros_in_session(_world: &mut ApplicationWorld) {
    // The Rust host has no registered macro commands yet.
}

#[when("the user selects the Macros command category")]
fn select_macros(world: &mut ApplicationWorld) {
    click_in_customize(world, "AllCommands");
    world.click_text("Macros");
    world.pump(Vec::new());
}

#[then("the available list shows only \"<Separator>\"")]
fn only_separator_in_source(world: &mut ApplicationWorld) {
    let separator = world.run_in_window("Customize", "<Separator>");
    let left = separator.minimum[0] - 8.;
    let top = separator.minimum[1] - 1.;
    let right = world.run_in_window("Customize", ">").minimum[0];
    let rows: Vec<_> = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            let [x, y] = run.center();
            x >= left && x < right && y >= top && y < top + 200.
        })
        .map(|run| run.text.as_str())
        .collect();
    assert_eq!(rows, ["<Separator>"]);
}

#[then("the custom panel list remains empty")]
fn target_is_empty(world: &mut ApplicationWorld) {
    assert_eq!(separator_count(world), 1);
}

#[when("the user double-clicks \"<Separator>\" in the available list")]
fn double_click_separator(world: &mut ApplicationWorld) {
    let center = world.run_in_window("Customize", "<Separator>").center();
    world.double_click(diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
}

#[then("a separator appears in the custom panel list")]
#[then("\"<Separator>\" remains in the available list")]
fn separator_in_both_lists(world: &mut ApplicationWorld) {
    assert_eq!(separator_count(world), 2);
}

fn separator_count(world: &ApplicationWorld) -> usize {
    let (position, size) = diagnostics::window_geometry("Customize").unwrap();
    world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == "<Separator>" && {
                let center = run.center();
                (0..2).all(|axis| {
                    center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis]
                })
            }
        })
        .count()
}
