// SPDX-License-Identifier: GPL-2.0-only
//! Hide and restore Explore from the expanded Modify title.
use super::super::ribbon_panel_visibility::ribbon_runs;
use super::{ApplicationWorld, diagnostics};
use cucumber::{then, when};

#[then("Explore is unchecked in the submenu")]
fn explore_unchecked(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(explore_row(world)));
}

#[when("the user selects Explore in the submenu")]
fn toggle_explore(world: &mut ApplicationWorld) {
    if world.work_features_x_before_explore_toggle.is_none() {
        expect_explore_caption(world);
        world.work_features_x_before_explore_toggle = Some(work_features_x(world));
    }
    let point = diagnostics::pointer_pixels(explore_row(world).center());
    world.click(point);
    world.pump(Vec::new());
}

#[then("Explore is hidden and the ribbon panels reflow")]
fn explore_hidden_and_reflowed(world: &mut ApplicationWorld) {
    assert!(
        ribbon_runs(world, "Explore").next().is_none(),
        "Explore remains visible"
    );
    assert!(work_features_x(world) < initial_work_features_x(world));
}

#[then("Explore returns to the 3D Model ribbon")]
fn explore_restored(world: &mut ApplicationWorld) {
    expect_explore_caption(world);
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

fn expect_explore_caption(world: &ApplicationWorld) {
    assert!(
        ribbon_runs(world, "Explore").next().is_some(),
        "expected the Explore ribbon caption"
    );
}

fn work_features_x(world: &ApplicationWorld) -> f32 {
    ribbon_runs(world, "Work Features")
        .next()
        .expect("expected a Work Features ribbon caption")
        .center()[0]
}

fn initial_work_features_x(world: &ApplicationWorld) -> f32 {
    world
        .work_features_x_before_explore_toggle
        .expect("recorded Work Features before hiding Explore")
}
