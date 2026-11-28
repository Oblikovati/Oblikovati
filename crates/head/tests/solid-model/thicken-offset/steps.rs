// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Modify Thicken/Offset scenarios (thicken-offset-01..05).
mod advanced;
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use setup::{
    EDIT_ERROR_WINDOW, THICKEN_WINDOW, accept_thicken_and_wait, click_thicken_control,
    double_click_feature, expect_ready_preview, open_thicken_ribbon, open_type_combo,
    replace_thicken_field, select_block_planar_face, select_exposed_circular_face, set_checkbox,
};

#[given("the two-body part has an exposed circular planar end face")]
fn two_body_has_circular_end_face(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 2, "expected 2 solid bodies");
}

#[when("the user opens Thicken/Offset and selects that face")]
fn open_thicken_and_select_circular_face(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    select_exposed_circular_face(world);
}

#[then("the panel displays 1 Face and enables OK")]
fn panel_displays_one_face_and_enables_ok(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(text.contains("1Face"), "expected 1 Face, saw {text}");
    expect_ready_preview(world);
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[when("the user chooses New Solid and accepts a 1 mm distance")]
fn choose_new_solid_and_accept_1mm(world: &mut ApplicationWorld) {
    click_thicken_control(world, "NewSolid");
    replace_thicken_field(world, "Distance", "1 mm");
    expect_ready_preview(world);
    accept_thicken_and_wait(world);
}

#[then("Thicken1 appears in the browser and the solid body count becomes three")]
fn thicken1_appears_and_three_solids(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Thicken1"),
        "expected Thicken1 in browser: {:?}",
        browser.features
    );
    assert_eq!(
        browser.bodies.len(),
        3,
        "expected 3 solid bodies, saw {:?}",
        browser.bodies
    );
}

#[given("the user creates Thicken1 as a 1 mm New Solid from the exposed circular planar end face")]
fn create_thicken1_new_solid(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    select_exposed_circular_face(world);
    click_thicken_control(world, "NewSolid");
    replace_thicken_field(world, "Distance", "1 mm");
    expect_ready_preview(world);
    accept_thicken_and_wait(world);
}

#[given("Thicken1 is open for editing with Allow Approximation enabled")]
fn reopen_thicken1_with_allow_approx(world: &mut ApplicationWorld) {
    double_click_feature(world, "Thicken1");
    let current = world
        .state()
        .thicken_offset_preview
        .is_some_and(|p| p.request.allow_approximation);
    set_checkbox(world, "AllowApproximation", current, true);
}

pub(in super::super) fn open_approx_type_dropdown_for_thicken(world: &mut ApplicationWorld) {
    open_type_combo(world);
}

#[then("the options include Never too thin, Never too thick, and Mean")]
fn approx_type_options_listed(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.text());
    for expected in ["Nevertoothin", "Nevertoothick", "Mean"] {
        assert!(
            text.contains(expected),
            "expected {expected} in Type dropdown, saw {text}"
        );
    }
    world.escape();
    world.pump(Vec::new());
}

#[when("the user chooses Specified tolerance")]
fn choose_specified_tolerance(world: &mut ApplicationWorld) {
    click_thicken_control(world, "Specified");
}

#[then("a percentage field appears with a red validation indicator and OK is disabled")]
fn percentage_field_invalid_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("Percent") && text.contains('!'),
        "expected percentage field with validation indicator, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[given("Thicken/Offset is open")]
fn thicken_offset_is_open(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
}

#[when("the user switches the output type control")]
#[when("the user switches to Offset Surface")]
fn switch_to_offset_surface(world: &mut ApplicationWorld) {
    click_thicken_control(world, "Surface");
}

#[then("the panel changes to OffsetSrf and displays a surface body name")]
fn panel_shows_offsetsrf_and_surface_name(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("OffsetSrf") && text.contains("Srf1"),
        "expected OffsetSrf and Srf1 in panel, saw {text}"
    );
}

#[when("the user selects a planar face and accepts a 1 mm offset")]
fn select_planar_face_and_accept_1mm_offset(world: &mut ApplicationWorld) {
    select_block_planar_face(world);
    replace_thicken_field(world, "Distance", "1 mm");
    expect_ready_preview(world);
    accept_thicken_and_wait(world);
}

#[then("OffsetSrf1 and Surface Bodies (1) appear in the browser")]
fn offsetsrf1_and_surface_bodies_appear(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "OffsetSrf1"),
        "expected OffsetSrf1 in browser: {:?}",
        browser.features
    );
    assert_eq!(
        browser.surface_bodies.len(),
        1,
        "expected 1 surface body, saw {:?}",
        browser.surface_bodies
    );
    world.expect_text("Surface Bodies (1)");
}

#[then("an offset surface is visible alongside the selected face")]
fn offset_surface_visible(world: &mut ApplicationWorld) {
    assert!(
        !world.app().window.scene_triangles.is_empty(),
        "expected offset surface triangles in scene"
    );
    let max_z = world
        .app()
        .window
        .scene_triangles
        .iter()
        .flat_map(|t| t.positions)
        .map(|p| p[2])
        .fold(f32::NEG_INFINITY, f32::max);
    assert!(
        max_z > 2.05,
        "expected offset surface extending past block top z=2.0 cm (to 2.1 cm), saw max_z={max_z}"
    );
}

#[given("Thicken/Offset is open on the part with one solid and a selectable planar face")]
fn open_thicken_on_single_solid(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1);
    open_thicken_ribbon(world);
}

#[when(
    "the user selects that face, enters 5 mm for Distance, selects the middle Direction and accepts the first Boolean option"
)]
fn select_face_5mm_middle_dir_join(world: &mut ApplicationWorld) {
    select_block_planar_face(world);
    replace_thicken_field(world, "Distance", "5 mm");
    click_thicken_control(world, "Dir2");
    click_thicken_control(world, "Join");
    expect_ready_preview(world);
    accept_thicken_and_wait(world);
}

#[then("Thicken1 appears and the solid body count remains one")]
fn thicken1_appears_and_count_remains_one(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Thicken1"));
    assert_eq!(browser.bodies.len(), 1);
}

#[when("the user reopens Thicken1")]
fn reopen_thicken1(world: &mut ApplicationWorld) {
    double_click_feature(world, "Thicken1");
}

#[then("the panel shows 1 Face and Distance 5 mm")]
fn panel_shows_one_face_and_5mm(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("1Face") && text.contains("5mm"),
        "expected 1 Face and 5 mm, saw {text}"
    );
}

#[given(
    "the user creates Thicken1 from a planar face at 5 mm using the middle Direction and first Boolean option"
)]
fn create_thicken1_5mm_join(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    select_face_5mm_middle_dir_join(world);
}

#[given("Thicken1 is open for editing")]
fn thicken1_open_for_editing(world: &mut ApplicationWorld) {
    double_click_feature(world, "Thicken1");
}

#[when("the user selects the second Boolean option and clicks OK")]
fn select_cut_and_click_ok(world: &mut ApplicationWorld) {
    click_thicken_control(world, "Cut");
    click_thicken_control(world, "OK");
    world.pump(Vec::new());
}

#[then("the MCAD Software reports a problem executing Edit Thicken feature")]
fn reports_edit_thicken_error(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(EDIT_ERROR_WINDOW).is_some(),
        "expected Edit Thicken modal, saw {}",
        world.text()
    );
    let modal = super::super::world::squeezed(&world.window_text(EDIT_ERROR_WINDOW));
    assert!(
        modal.contains("EditThicken"),
        "expected Edit Thicken error message, saw {modal}"
    );
}

#[when("the user cancels the error")]
fn cancel_edit_thicken_error(world: &mut ApplicationWorld) {
    let center = world.run_in_window(EDIT_ERROR_WINDOW, "Cancel").center();
    world.click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("the model view returns with Thicken1 in the browser")]
fn model_returns_with_thicken1(world: &mut ApplicationWorld) {
    assert!(super::super::diagnostics::window_geometry(EDIT_ERROR_WINDOW).is_none());
    assert!(super::super::diagnostics::window_geometry(THICKEN_WINDOW).is_none());
    let browser = world.state().browser.expect("expected browser");
    assert!(browser.features.iter().any(|f| f.name == "Thicken1"));
    assert!(!world.app().window.scene_triangles.is_empty());
}
