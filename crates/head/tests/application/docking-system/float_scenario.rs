// SPDX-License-Identifier: GPL-2.0-only
//! Exercise floating and redocking through native pointer input.
use super::{ApplicationWorld, PANEL, diagnostics, native_drag, viewport};
use cucumber::{then, when};

#[when("the user drags its title bar away from the dock targets")]
fn float_docked_panel(world: &mut ApplicationWorld) {
    let (minimum, maximum) = diagnostics::window_dock_tab_bounds(PANEL).unwrap();
    let start = [
        ((minimum[0] + maximum[0]) * 0.5) as i32,
        ((minimum[1] + maximum[1]) * 0.5) as i32,
    ];
    let area = viewport(world);
    let target = [
        ((area[0] + area[2]) / 2) as i32,
        ((area[1] + area[3]) / 2) as i32,
    ];
    native_drag(world, start, target);
}

#[then("the panel floats above the workspace")]
fn panel_is_floating(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_id(PANEL), Some(0));
    world.run_in_window(PANEL, "Assigntopart");
    world.save_capture("material-browser-floating");
}

#[then("the viewport expands into the space released by the panel")]
fn viewport_expands(world: &mut ApplicationWorld) {
    let (docked, _) = world.docking_resize.expect("docked viewport");
    let floating = viewport(world);
    assert!(
        floating[2] > docked[2],
        "viewport {docked:?} -> {floating:?}"
    );
}

#[when("the user drags the floating panel to a workspace dock target")]
fn redock_floating_panel(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(PANEL).unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    let area = viewport(world);
    let target = [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32];
    native_drag(world, start, target);
}

#[then("the panel docks at that target")]
fn panel_is_redocked(world: &mut ApplicationWorld) {
    assert_ne!(diagnostics::window_dock_id(PANEL).unwrap(), 0);
    let [_, _, right, _] = diagnostics::window_pixel_bounds(PANEL).expect("redocked panel");
    let width = world.extent()[0] as f32;
    assert!(
        (right - width).abs() <= 1.,
        "right dock edge {right}, window {width}"
    );
}

#[then("the viewport adjusts to the resulting workspace layout")]
fn viewport_adjusts(world: &mut ApplicationWorld) {
    let [left, _, right, _] = viewport(world).map(|edge| edge as f32);
    let [_, _, model_right, _] = diagnostics::window_pixel_bounds("Model").expect("Model dock");
    let [material_left, _, _, _] = diagnostics::window_pixel_bounds(PANEL).expect("Material dock");
    assert!(
        left < right
            && (0. ..=3.).contains(&(left - model_right))
            && (0. ..=3.).contains(&(material_left - right)),
        "viewport {left}..{right} must fill the space between docks at {model_right} and {material_left}"
    );
    world.run_in_window(PANEL, "Assigntopart");
    world.expect_text("End of Part");
    assert_eq!(world.state().document, world.previous_document);
    super::workspace_layout::browser_feature::expect_solid_visible(world);
    world.save_capture("material-browser-redocked");
}
