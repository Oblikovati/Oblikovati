// SPDX-License-Identifier: GPL-2.0-only
//! Hide and restore Explore from the expanded Modify title.
use super::{ApplicationWorld, diagnostics};
use cucumber::{then, when};

const RIBBON: &str = "##application-ribbon";

#[then("Explore is unchecked in the submenu")]
fn explore_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(explore_row(world)));
}

#[when("the user selects Explore in the submenu")]
fn toggle_explore(world: &mut ApplicationWorld) {
    if world.work_features_x_before_explore_toggle.is_none() {
        world.run_in_window(RIBBON, "Explore");
        world.work_features_x_before_explore_toggle = Some(work_features_x(world));
    }
    let point = diagnostics::pointer_pixels(explore_row(world).center());
    world.click(point);
    world.pump(Vec::new());
}

#[then("Explore is hidden and the ribbon panels reflow")]
fn explore_hidden_and_reflowed(world: &mut ApplicationWorld) {
    world.expect_no_text("Explore");
    assert!(work_features_x(world) < initial_work_features_x(world));
}

#[then("Explore returns to the 3D Model ribbon")]
fn explore_restored(world: &mut ApplicationWorld) {
    world.run_in_window(RIBBON, "Explore");
    assert_eq!(work_features_x(world), initial_work_features_x(world));
}

fn explore_row(world: &ApplicationWorld) -> &diagnostics::RenderedRun {
    let list = world.run("Work Features").list;
    world
        .frame
        .runs
        .iter()
        .find(|run| run.list == list && run.text == "Explore")
        .expect("Explore in the Show Panels submenu")
}

fn work_features_x(world: &ApplicationWorld) -> f32 {
    world.run_in_window(RIBBON, "WorkFeatures").center()[0]
}

fn initial_work_features_x(world: &ApplicationWorld) -> f32 {
    world
        .work_features_x_before_explore_toggle
        .expect("recorded Work Features before hiding Explore")
}
