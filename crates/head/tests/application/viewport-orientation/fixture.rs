// SPDX-License-Identifier: GPL-2.0-only
//! Open the same saved part for every current orientation scenario.
use super::*;

const ORIENTATION_PART: &str = "fixtures/cad/docking-visible-solid-current.opd";

#[given("`fixtures/cad/docking-visible-solid-current.opd` is active in the MCAD Software")]
#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is active with its rectangular block visible in the MCAD Software"
)]
fn open_orientation_part(world: &mut ApplicationWorld) {
    let path = super::super::scenario_root().join(ORIENTATION_PART);
    world.open_fixture(&path);
    let document = world.active_id();
    let state = world.state();
    let storage = &state.document_storage[&document];
    assert_eq!(
        storage.path.as_deref().map(std::path::Path::new),
        Some(path.as_path())
    );
    let browser = state.browser.expect("expected the part model browser");
    assert_eq!(browser.bodies.len(), 1);
    assert!(browser.bodies[0].visible, "expected the rectangular solid");
    if browser.sketches.iter().any(|sketch| sketch.visible) {
        hide_profile_overlay(world);
    }
    solid_extent(world);
    world.parts.push(document);
}

fn hide_profile_overlay(world: &mut ApplicationWorld) {
    use super::super::workspace_layout::browser_feature::{choose_menu_row, right_click_row};
    right_click_row(world, "Profile");
    choose_menu_row(world, "Visibility");
    world.pump(Vec::new());
    super::behaviour::nothing_selected(world);
    let browser = world
        .state()
        .browser
        .expect("expected the part model browser");
    assert!(
        !browser.sketches[0].visible,
        "the sketch dimensions must clear the solid observation"
    );
}

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is active in the MCAD Software with its Top view displayed"
)]
#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is active with its rectangular block visible in the MCAD Software in Top view"
)]
fn orientation_part_top(world: &mut ApplicationWorld) {
    open_orientation_part(world);
    press_part(world, |parts| parts.home);
    press_top_in_iso(world);
    expect_control_shows(world, "Top");
}

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is active with its rectangular block visible in the MCAD Software in upright Front view"
)]
fn orientation_part_upright(world: &mut ApplicationWorld) {
    open_orientation_part(world);
    front_displayed(world);
}

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is active with the block visible in an oblique view"
)]
fn orientation_part_oblique(world: &mut ApplicationWorld) {
    open_orientation_part(world);
    press_part(world, |parts| parts.home);
    expect_control_shows(world, "TopFrontRight");
}
