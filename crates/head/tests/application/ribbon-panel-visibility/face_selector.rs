// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::super::{
    ribbon_keyboard_navigation::visible_face_point, scenario_root,
    viewport_orientation::open_saved_navigation_block,
};
use super::*;

#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open with a visible solid in the MCAD Software"
)]
#[given(
    "`fixtures/cad/docking-visible-solid-current.opd` is open with its solid block in the MCAD Software"
)]
fn visible_named_solid(world: &mut ApplicationWorld) {
    let path = scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    open_saved_navigation_block(world, &path);
}

#[given("the full Tools ribbon and its panel selector are open")]
fn visible_tools_selector(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Application Options");
    open_tools_selector(world);
}

#[when("the user right-clicks a visible solid face outside the selector")]
fn right_click_face(world: &mut ApplicationWorld) {
    let point = visible_face_point(world);
    world.observed_view = Some(point);
    world.right_click(point);
}

#[then("no face context menu is visible in the captured state")]
fn no_face_context(world: &mut ApplicationWorld) {
    for label in ["Select Tangencies", "Export Face As...", "New Sketch"] {
        world.expect_no_text(label);
    }
}

#[then("a small face-selection control appears near the pointer")]
fn toolbar_appears_near_pick(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    assert!(world.state().model_face_chosen);
    let toolbar = "##selection-toolbar";
    world.run_in_window(toolbar, "ZoomSelected");
    let (position, size) = diagnostics::window_geometry(toolbar).expect("expected face toolbar");
    let minimum = diagnostics::pointer_pixels(position);
    let maximum = diagnostics::pointer_pixels([position[0] + size[0], position[1] + size[1]]);
    let point = world.observed_view.expect("expected the right-click point");
    for axis in 0..2 {
        assert!((minimum[axis] - 24..=maximum[axis] + 24).contains(&point[axis]));
    }
}
