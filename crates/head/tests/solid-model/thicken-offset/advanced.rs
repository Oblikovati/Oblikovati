// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Thicken/Offset scenarios 06..09 (zero/negative distance, tolerance, face chain).
use super::super::super::*;
use super::setup::{
    THICKEN_WINDOW, accept_thicken_and_wait, click_thicken_control, double_click_feature,
    expect_ready_preview, open_thicken_ribbon, replace_thicken_field, select_block_planar_face,
    select_cylindrical_side_face, select_rounded_cylinder_side_face, set_checkbox,
};
use cucumber::{given, then, when};
use oblikovati_model::feature::thicken_offset::{
    ThickenApproximationTolerance, ThickenBoolean, ThickenDirection, ThickenOutputMode,
};

#[given("a planar face is selected in the Thicken/Offset properties")]
fn planar_face_selected_in_thicken(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    select_block_planar_face(world);
}

#[when("the user enters 0 as the distance in solid Thicken mode")]
fn enter_zero_distance_solid(world: &mut ApplicationWorld) {
    replace_thicken_field(world, "Distance", "0");
}

#[then("the properties show a validation error and OK is disabled")]
#[then("the field shows a validation indicator and OK is disabled")]
fn validation_error_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains('!'),
        "expected validation indicator, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[then("the selected face and 0 mm distance remain and OK is enabled")]
fn face_and_zero_remain_ok_enabled(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected preview");
    assert_eq!(preview.request.output, ThickenOutputMode::Surface);
    assert_eq!(preview.request.faces.len(), 1);
    assert_eq!(preview.request.distance, Some(0.0));
    expect_ready_preview(world);
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[when("the user accepts and reopens the resulting OffsetSrf1 feature")]
#[when("the user accepts and reopens OffsetSrf1")]
fn accept_and_reopen_offsetsrf1(world: &mut ApplicationWorld) {
    accept_thicken_and_wait(world);
    double_click_feature(world, "OffsetSrf1");
}

#[then("the properties show 1 Face and a distance of 0 mm")]
fn properties_show_one_face_zero_mm(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("1Face") && text.contains("0mm"),
        "expected 1 Face and 0 mm, saw {text}"
    );
}

#[given("the user creates OffsetSrf1 by offsetting a planar face by 1 mm")]
fn create_offsetsrf1_1mm(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    click_thicken_control(world, "Surface");
    select_block_planar_face(world);
    replace_thicken_field(world, "Distance", "1 mm");
    expect_ready_preview(world);
    accept_thicken_and_wait(world);
}

#[given(
    "OffsetSrf1 is open for editing with Allow Approximation enabled and Optimized tolerance selected"
)]
fn reopen_offsetsrf1_with_optimized_approx(world: &mut ApplicationWorld) {
    double_click_feature(world, "OffsetSrf1");
    let current = world
        .state()
        .thicken_offset_preview
        .is_some_and(|p| p.request.allow_approximation);
    set_checkbox(world, "AllowApproximation", current, true);
    click_thicken_control(world, "Optimized");
}

#[when("the user chooses Specified and replaces the displayed value with 1")]
fn choose_specified_and_enter_1(world: &mut ApplicationWorld) {
    click_thicken_control(world, "Specified");
    replace_thicken_field(world, "Percent", "1");
}

#[then("the validation indicator clears and OK becomes enabled")]
fn validation_clears_and_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        !text.contains('!'),
        "expected validation indicator cleared, saw {text}"
    );
    expect_ready_preview(world);
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[when("the user clicks OK and reopens OffsetSrf1")]
fn click_ok_and_reopen_offsetsrf1(world: &mut ApplicationWorld) {
    accept_thicken_and_wait(world);
    double_click_feature(world, "OffsetSrf1");
}

#[then("Specified and 1,0 % are displayed with a validation indicator and OK disabled")]
fn specified_one_comma_zero_percent_invalid(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected preview");
    assert_eq!(
        preview.request.approximation_tolerance,
        ThickenApproximationTolerance::Specified
    );
    assert!(!preview.request.approximation_valid);
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("Specified") && text.contains("1,0%") && text.contains('!'),
        "expected Specified, 1,0 %, and !, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("the user turns off Allow Approximation")]
fn turn_off_allow_approximation(world: &mut ApplicationWorld) {
    let current = world
        .state()
        .thicken_offset_preview
        .is_some_and(|p| p.request.allow_approximation);
    set_checkbox(world, "AllowApproximation", current, false);
}

#[then("the Type and Tolerance controls disappear and OK is enabled")]
fn type_and_tolerance_disappear_ok_enabled(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        !text.contains("Type") && !text.contains("Tolerance"),
        "expected Type and Tolerance hidden, saw {text}"
    );
    expect_ready_preview(world);
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[given("the cylindrical side face is selected in solid Thicken")]
fn cylindrical_side_selected_in_solid_thicken(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    select_cylindrical_side_face(world);
}

#[when("the user enters -1 for Distance")]
fn enter_negative_distance(world: &mut ApplicationWorld) {
    replace_thicken_field(world, "Distance", "-1");
}

#[then("the negative value and validation indicator remain and OK is disabled")]
fn negative_value_and_indicator_remain(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("-1") && text.contains('!'),
        "expected -1 and validation indicator, saw {text}"
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("the user replaces the value with 1 and chooses the first direction")]
fn replace_with_1_and_choose_first_direction(world: &mut ApplicationWorld) {
    replace_thicken_field(world, "Distance", "1");
    click_thicken_control(world, "Dir1");
    expect_ready_preview(world);
}

#[then("the panel shows 1 Face, 1 mm, and the first direction selected")]
fn panel_shows_one_face_1mm_first_dir(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected preview");
    assert_eq!(preview.request.faces.len(), 1);
    assert_eq!(preview.request.direction, ThickenDirection::Negative);
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("1Face") && text.contains("1mm"),
        "expected 1 Face and 1 mm, saw {text}"
    );
}

#[given("the part has one rounded body and two separate surface bodies")]
#[given("the browser lists one solid body and two surface bodies")]
fn verify_one_solid_two_surfaces(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(browser.bodies.len(), 1, "expected 1 solid body");
    assert_eq!(browser.surface_bodies.len(), 2, "expected 2 surface bodies");
}

#[given("the user opens solid Thicken with a distance of 1 mm")]
fn open_solid_thicken_1mm(world: &mut ApplicationWorld) {
    open_thicken_ribbon(world);
    click_thicken_control(world, "Solid");
    replace_thicken_field(world, "Distance", "1 mm");
}

#[given("Allow Approximation and Automatic Face Chain are enabled before selection")]
fn enable_approx_and_face_chain(world: &mut ApplicationWorld) {
    let preview = world.state().thicken_offset_preview;
    let approx = preview
        .as_ref()
        .is_some_and(|p| p.request.allow_approximation);
    let chain = preview
        .as_ref()
        .is_some_and(|p| p.request.automatic_face_chain);
    set_checkbox(world, "AllowApproximation", approx, true);
    set_checkbox(world, "AutomaticFaceChain", chain, true);
}

#[given("the second direction is selected")]
fn select_second_direction(world: &mut ApplicationWorld) {
    click_thicken_control(world, "Dir2");
}

#[when("the user clicks the cylindrical side face adjacent to the rounded end")]
fn click_rounded_cylinder_side_face(world: &mut ApplicationWorld) {
    select_rounded_cylinder_side_face(world);
}

#[then("the selector displays 2 Faces")]
fn selector_displays_two_faces(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(text.contains("2Faces"), "expected 2 Faces, saw {text}");
}

#[when("the user enables Automatic Blending")]
fn enable_automatic_blending(world: &mut ApplicationWorld) {
    let current = world
        .state()
        .thicken_offset_preview
        .is_some_and(|p| p.request.automatic_blending);
    set_checkbox(world, "AutomaticBlending", current, true);
}

#[then("the Boolean output controls and third direction choice are hidden")]
fn boolean_and_third_direction_hidden(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        !text.contains("Symmetric") && !text.contains("NewSolid") && !text.contains("Intersect"),
        "expected Boolean controls and Symmetric hidden, saw {text}"
    );
}

#[when("the user accepts and reopens Thicken1")]
fn accept_and_reopen_thicken1(world: &mut ApplicationWorld) {
    expect_ready_preview(world);
    accept_thicken_and_wait(world);
    double_click_feature(world, "Thicken1");
}

#[then("the panel shows 2 Faces, 1 mm, and Automatic Blending enabled")]
fn panel_shows_two_faces_1mm_blending(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected preview");
    assert_eq!(preview.request.faces.len(), 2);
    assert!(preview.request.automatic_blending);
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("2Faces") && text.contains("1mm"),
        "expected 2 Faces and 1 mm, saw {text}"
    );
}

#[when("the user disables Automatic Blending")]
fn disable_automatic_blending(world: &mut ApplicationWorld) {
    let current = world
        .state()
        .thicken_offset_preview
        .is_some_and(|p| p.request.automatic_blending);
    set_checkbox(world, "AutomaticBlending", current, false);
}

#[then("the Boolean controls and third direction choice reappear")]
fn boolean_and_third_direction_reappear(world: &mut ApplicationWorld) {
    let text = super::super::super::world::squeezed(&world.window_text(THICKEN_WINDOW));
    assert!(
        text.contains("Symmetric")
            && text.contains("Join")
            && text.contains("Cut")
            && text.contains("NewSolid"),
        "expected Boolean controls and Symmetric restored, saw {text}"
    );
    let preview = world
        .state()
        .thicken_offset_preview
        .expect("expected preview");
    assert_eq!(preview.request.boolean, ThickenBoolean::Join);
}
