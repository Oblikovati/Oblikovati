// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Surface Ruled Surface scenarios (ruled-surface-01..10).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiRuledSurfacePreview;
use oblikovati_model::feature::ruled_surface::{RuledSurfaceType, RuledSurfaceVector};
use setup::{
    RULED_WINDOW, choose_origin_z_axis, click_ruled_control, create_1mm_offset_surface,
    double_click_ruled_feature, expect_ready_preview, open_ruled_surface_ribbon,
    pick_offset_surface_edge, previous_arrow_direction, previous_surface_count,
    record_arrow_direction, record_surface_count, replace_field_in, set_automatic_edge_chain,
};

#[given("the user creates a 1 mm offset surface from the top planar face")]
fn create_offset_surface_step(world: &mut ApplicationWorld) {
    create_1mm_offset_surface(world);
    record_surface_count(world);
}

#[given("Ruled Surface is open with Automatic Edge Chain checked")]
fn ruled_surface_open_chain_checked(world: &mut ApplicationWorld) {
    open_ruled_surface_ribbon(world);
    set_automatic_edge_chain(world, true);
}

#[when("the user clicks one edge of the rectangular surface")]
fn click_one_rectangular_surface_edge(world: &mut ApplicationWorld) {
    pick_offset_surface_edge(world, 4);
}

#[then("the selection list contains its four edges and Apply is available.")]
fn selection_list_has_four_edges_and_apply_enabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    for edge_label in ["Edge1", "Edge2", "Edge3", "Edge4"] {
        assert!(
            text.contains(edge_label),
            "expected {edge_label} in Ruled Surface selection list, saw {text}"
        );
    }
    expect_ready_preview(world);
    assert!(world.run_is_enabled("Apply"), "expected Apply enabled");
}

#[given(
    "Ruled Surface is open in the first mode with Automatic Edge Chain selecting all four offset-surface edges"
)]
fn ruled_surface_open_first_mode_all_four_edges(world: &mut ApplicationWorld) {
    open_ruled_surface_ribbon(world);
    click_ruled_control(world, "Normal");
    set_automatic_edge_chain(world, true);
    pick_offset_surface_edge(world, 4);
    expect_ready_preview(world);
    record_surface_count(world);
}

#[then("a surface body is added, the selection is cleared, and the dialog remains open.")]
fn surface_body_added_selection_cleared_dialog_open(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.surface_bodies.len(),
        previous_surface_count() + 1,
        "expected a new surface body, saw {:?}",
        browser.surface_bodies
    );
    assert!(
        super::super::diagnostics::window_geometry(RULED_WINDOW).is_some(),
        "expected Ruled Surface dialog to remain open after Apply"
    );
    let text = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    assert!(
        !text.contains("Edge1"),
        "expected selection list cleared after Apply, saw {text}"
    );
}

#[given("Ruled Surface is open in the third mode with Automatic Edge Chain unchecked")]
fn ruled_surface_open_third_mode_no_chain(world: &mut ApplicationWorld) {
    open_ruled_surface_ribbon(world);
    click_ruled_control(world, "Sweep");
    set_automatic_edge_chain(world, false);
}

#[when("the user selects one edge")]
fn user_selects_one_edge(world: &mut ApplicationWorld) {
    pick_offset_surface_edge(world, 1);
}

#[then("the list contains one edge and Apply is disabled")]
fn list_has_one_edge_and_apply_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    assert!(
        text.contains("Edge1") && !text.contains("Edge2"),
        "expected exactly Edge1 in list, saw {text}"
    );
    assert!(
        !world.run_is_enabled("Apply"),
        "expected Apply disabled before Vector is chosen"
    );
}

#[when("the user chooses the Z Axis as Vector")]
fn user_chooses_z_axis_as_vector(world: &mut ApplicationWorld) {
    choose_origin_z_axis(world);
}

#[then("a direction arrow appears and Apply becomes available.")]
fn direction_arrow_appears_and_apply_available(world: &mut ApplicationWorld) {
    expect_ready_preview(world);
    let preview = world
        .state()
        .ruled_surface_preview
        .expect("expected ruled surface preview");
    let arrow = preview
        .arrow
        .expect("expected direction arrow after choosing Z Axis");
    let origin_px = crate::HeadWindow::model_pixel(&world.app().window.native, arrow.origin_world)
        .expect("expected direction arrow origin to project on screen");
    let tip_px = crate::HeadWindow::model_pixel(&world.app().window.native, arrow.tip_world)
        .expect("expected direction arrow tip to project on screen");
    assert_ne!(
        origin_px, tip_px,
        "expected visible non-zero projected direction arrow, got {arrow:?}"
    );
    assert!(
        arrow.direction[2] > 0.99,
        "expected Z Axis direction arrow, got {arrow:?}"
    );
    assert!(world.run_is_enabled("Apply"), "expected Apply enabled");
}

#[given("the user applies a first-mode ruled surface using the offset-surface edge chain")]
fn user_applies_first_mode_ruled_surface(world: &mut ApplicationWorld) {
    open_ruled_surface_ribbon(world);
    click_ruled_control(world, "Normal");
    set_automatic_edge_chain(world, true);
    pick_offset_surface_edge(world, 4);
    expect_ready_preview(world);
    let before = world.state().browser.unwrap().surface_bodies.len();
    click_ruled_control(world, "Apply");
    world.pump(Vec::new());
    let after = world.state().browser.unwrap().surface_bodies.len();
    assert_eq!(after, before + 1, "expected applied surface body");
    record_surface_count(world);
}

#[given(
    "the user selects one offset-surface edge in the third mode with the Z Axis as Vector and a preview is present"
)]
fn select_one_edge_third_mode_z_axis_preview(world: &mut ApplicationWorld) {
    click_ruled_control(world, "Sweep");
    set_automatic_edge_chain(world, false);
    pick_offset_surface_edge(world, 1);
    choose_origin_z_axis(world);
    expect_ready_preview(world);
    let arrow = world
        .state()
        .ruled_surface_preview
        .and_then(|p| p.arrow)
        .expect("expected direction arrow before flip");
    record_arrow_direction(arrow.direction);
}

#[when("the user reverses its direction")]
fn user_reverses_direction(world: &mut ApplicationWorld) {
    click_ruled_control(world, "Flip");
}

#[then("the direction arrow reverses")]
fn direction_arrow_reverses(world: &mut ApplicationWorld) {
    expect_ready_preview(world);
    let prev = previous_arrow_direction().expect("expected previous arrow direction");
    let arrow = world
        .state()
        .ruled_surface_preview
        .and_then(|p| p.arrow)
        .expect("expected flipped direction arrow");
    let dot =
        prev[0] * arrow.direction[0] + prev[1] * arrow.direction[1] + prev[2] * arrow.direction[2];
    assert!(
        dot < -0.99 && arrow.flipped,
        "expected reversed direction arrow: prev={prev:?}, now={arrow:?}"
    );
}

#[then("the dialog closes and the previously applied surface remains.")]
fn dialog_closes_and_applied_surface_remains(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry(RULED_WINDOW).is_none(),
        "expected Ruled Surface dialog to close after Cancel"
    );
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.surface_bodies.len(),
        previous_surface_count(),
        "expected previously applied surface body to remain, saw {:?}",
        browser.surface_bodies
    );
}

#[given("Ruled Surface is open in the second mode with Automatic Edge Chain unchecked")]
fn ruled_surface_open_second_mode_no_chain(world: &mut ApplicationWorld) {
    open_ruled_surface_ribbon(world);
    click_ruled_control(world, "Tangent");
    set_automatic_edge_chain(world, false);
    record_surface_count(world);
}

#[when("the user selects one edge and clicks Apply")]
fn select_one_edge_and_apply(world: &mut ApplicationWorld) {
    pick_offset_surface_edge(world, 1);
    expect_ready_preview(world);
    click_ruled_control(world, "Apply");
    world.pump(Vec::new());
}

#[then("another surface body is added and the selection clears.")]
fn another_surface_body_added_and_selection_clears(world: &mut ApplicationWorld) {
    surface_body_added_selection_cleared_dialog_open(world);
}

#[given(
    "Ruled Surface is open in the third mode with one offset-surface edge and the Z Axis selected"
)]
fn ruled_surface_open_third_mode_one_edge_z_axis(world: &mut ApplicationWorld) {
    open_ruled_surface_ribbon(world);
    click_ruled_control(world, "Sweep");
    set_automatic_edge_chain(world, false);
    pick_offset_surface_edge(world, 1);
    choose_origin_z_axis(world);
    expect_ready_preview(world);
    record_surface_count(world);
}

#[when("the user replaces Distance with 20 and clicks Apply")]
fn replace_distance_20_and_apply(world: &mut ApplicationWorld) {
    replace_field_in(world, RULED_WINDOW, "Distance", "20");
    expect_ready_preview(world);
    click_ruled_control(world, "Apply");
    world.pump(Vec::new());
}

#[then("another surface body is added and the dialog resets for another selection.")]
fn another_surface_added_and_dialog_resets(world: &mut ApplicationWorld) {
    surface_body_added_selection_cleared_dialog_open(world);
    let preview = world.state().ruled_surface_preview;
    assert!(
        preview
            .as_ref()
            .is_none_or(|p| p.request.edges.is_empty() && p.request.vector.is_none()),
        "expected dialog reset for another selection, saw {preview:?}"
    );
}

#[given(
    "the user creates Ruled Surface1 from one offset-surface edge in the third mode using the Z Axis and 20 mm distance"
)]
fn create_ruled_surface1_sweep_z_20mm(world: &mut ApplicationWorld) {
    open_ruled_surface_ribbon(world);
    click_ruled_control(world, "Sweep");
    set_automatic_edge_chain(world, false);
    pick_offset_surface_edge(world, 1);
    choose_origin_z_axis(world);
    replace_field_in(world, RULED_WINDOW, "Distance", "20");
    expect_ready_preview(world);
    click_ruled_control(world, "OK");
    world.pump(Vec::new());
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser.features.iter().any(|f| f.name == "Ruled Surface1"),
        "expected Ruled Surface1 in browser, saw {:?}",
        browser.features
    );
}

#[when("the user double-clicks Ruled Surface1 in the model browser")]
#[when("the user reopens Ruled Surface1")]
fn double_click_ruled_surface1(world: &mut ApplicationWorld) {
    double_click_ruled_feature(world, "Ruled Surface1");
}

#[then("the edit dialog shows the third mode, Edge1, a 20 mm distance, and a 0 degree angle.")]
fn edit_dialog_shows_third_mode_edge1_20mm_0deg(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .ruled_surface_preview
        .expect("expected edit preview");
    assert_eq!(preview.request.ruled_type, RuledSurfaceType::Sweep);
    assert_eq!(preview.request.edges.len(), 1);
    assert_eq!(preview.request.vector, Some(RuledSurfaceVector::OriginZ));
    assert!((preview.request.distance.unwrap() - 2.0).abs() < 1.0e-6);
    assert!((preview.request.angle_deg.unwrap() - 0.0).abs() < 1.0e-6);
    let text = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    assert!(
        text.contains("Edge1") && text.contains("20mm") && text.contains("0deg"),
        "expected Edge1, 20 mm, 0 deg in edit dialog, saw {text}"
    );
}

#[given("Ruled Surface1 is open for editing")]
#[given("Ruled Surface1 is open for editing with its existing edge and vector")]
#[given("Ruled Surface1 is open for editing with its existing edge, vector and 20 mm distance")]
fn ruled_surface1_open_for_editing(world: &mut ApplicationWorld) {
    double_click_ruled_feature(world, "Ruled Surface1");
    expect_ready_preview(world);
}

#[when("the user changes Angle to 15 degrees and clicks OK")]
fn change_angle_to_15_and_ok(world: &mut ApplicationWorld) {
    replace_field_in(world, RULED_WINDOW, "Angle", "15");
    expect_ready_preview(world);
    click_ruled_control(world, "OK");
    world.pump(Vec::new());
    assert!(
        super::super::diagnostics::window_geometry(RULED_WINDOW).is_none(),
        "expected Ruled Surface dialog to close after OK"
    );
}

#[then("the dialog shows Angle 15 degrees and Distance 20 mm.")]
fn dialog_shows_angle_15_and_distance_20(world: &mut ApplicationWorld) {
    let preview = world
        .state()
        .ruled_surface_preview
        .expect("expected reopened preview");
    assert!((preview.request.angle_deg.unwrap() - 15.0).abs() < 1.0e-6);
    assert!((preview.request.distance.unwrap() - 2.0).abs() < 1.0e-6);
    let text = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    assert!(
        text.contains("15deg") && text.contains("20mm"),
        "expected 15 deg and 20 mm in dialog, saw {text}"
    );
}

#[when("the user enters 0 or -1 as Distance")]
fn enter_zero_or_negative_distance(world: &mut ApplicationWorld) {
    replace_field_in(world, RULED_WINDOW, "Distance", "0");
    let text_zero = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    assert!(
        text_zero.contains('!') && !world.run_is_enabled("OK"),
        "expected 0 distance to show red validation and disable OK, saw {text_zero}"
    );
    replace_field_in(world, RULED_WINDOW, "Distance", "-1");
}

#[then("the value is red and OK is disabled")]
fn distance_value_red_and_ok_disabled(world: &mut ApplicationWorld) {
    let text = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    assert!(
        text.contains('!'),
        "expected red validation indicator for invalid distance, saw {text}"
    );
    assert!(
        world.state().ruled_surface_preview.as_ref().is_some_and(
            |p| p.request.distance.is_none() && p.result == GuiRuledSurfacePreview::Incomplete
        ),
        "expected invalid distance preview to be Incomplete, saw {:?}",
        world.state().ruled_surface_preview
    );
    assert!(!world.run_is_enabled("OK"), "expected OK disabled");
}

#[when("the user replaces it with 1")]
fn replace_distance_with_1(world: &mut ApplicationWorld) {
    replace_field_in(world, RULED_WINDOW, "Distance", "1");
}

#[then("OK is enabled.")]
fn ok_is_enabled_period(world: &mut ApplicationWorld) {
    expect_ready_preview(world);
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}

#[when("the user enters 180 in Angle and leave the field")]
fn enter_180_in_angle_and_leave(world: &mut ApplicationWorld) {
    replace_field_in(world, RULED_WINDOW, "Angle", "180");
}

#[then("Angle displays 180 and OK remains enabled.")]
fn angle_displays_180_and_ok_enabled(world: &mut ApplicationWorld) {
    expect_ready_preview(world);
    let preview = world
        .state()
        .ruled_surface_preview
        .expect("expected preview after 180 angle");
    assert!((preview.request.angle_deg.unwrap() - 180.0).abs() < 1.0e-6);
    let text = super::super::world::squeezed(&world.window_text(RULED_WINDOW));
    assert!(
        text.contains("180"),
        "expected Angle to display 180, saw {text}"
    );
    assert!(world.run_is_enabled("OK"), "expected OK enabled");
}
