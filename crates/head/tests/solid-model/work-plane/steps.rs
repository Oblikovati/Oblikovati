// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Solid Model Work Plane (work-plane-01..12, #3918).

mod setup;

use super::super::*;
use cucumber::{given, then, when};
use oblikovati_viewport::ViewportPhysicalKey;
use setup::{
    base_edge_midpoint_near_origin_neg_x, choose_plane_menu_item, click_model_point,
    ensure_origin_expanded, orient_isometric_for_picking, planar_top_face_point,
    three_non_collinear_vertex_points, two_adjoining_edge_points, work_plane_count,
};

#[given("the rectangular solid has a selectable planar top face")]
fn rectangular_solid_has_top_face(world: &mut ApplicationWorld) {
    orient_isometric_for_picking(world);
    let _ = planar_top_face_point(world);
}

#[when("the user chooses Offset from Plane and selects that face")]
fn choose_offset_and_select_top_face(world: &mut ApplicationWorld) {
    choose_plane_menu_item(world, "Offset from Plane");
    let top = planar_top_face_point(world);
    click_model_point(world, top);
}

#[then("a distance field and plane preview appear")]
fn distance_field_and_plane_preview_appear(world: &mut ApplicationWorld) {
    world.expect_text("Offset Plane");
    world.expect_text("Offset");
    world.expect_text("1 Plane");
    assert!(
        world.state().work_plane_preview_active,
        "expected active live Work Plane preview"
    );
}

#[when("the user enters 5 mm and presses Enter")]
fn enter_5_mm_and_press_enter(world: &mut ApplicationWorld) {
    world.click_text("10 mm");
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("5 mm");
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("Work Plane1 appears in the model browser")]
fn work_plane_1_appears(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Work Plane1" && feature.kind == "Work Plane"),
        "expected Work Plane1 in model browser, saw {:?}",
        browser.features
    );
    world.expect_text("Work Plane1");
}

#[given("the user creates a work plane offset 5 mm from the planar top face")]
fn create_offset_work_plane_5mm(world: &mut ApplicationWorld) {
    orient_isometric_for_picking(world);
    choose_offset_and_select_top_face(world);
    enter_5_mm_and_press_enter(world);
    work_plane_1_appears(world);
}

#[when("the user chooses Midplane between Two Planes")]
fn choose_midplane_between_two_planes(world: &mut ApplicationWorld) {
    choose_plane_menu_item(world, "Midplane between Two Planes");
}

#[when("the user selects the work plane and the planar face")]
fn select_work_plane_and_planar_face(world: &mut ApplicationWorld) {
    world.click_text("Work Plane1");
    world.pump(Vec::new());
    let top = planar_top_face_point(world);
    click_model_point(world, top);
}

#[then("a second work plane appears in the model browser")]
fn second_work_plane_appears(world: &mut ApplicationWorld) {
    assert_eq!(
        work_plane_count(world),
        2,
        "expected 2 work planes in model browser"
    );
    world.expect_text("Work Plane2");
}

#[given("the user chooses Offset from Plane")]
fn given_choose_offset_from_plane(world: &mut ApplicationWorld) {
    world.observed_counts = vec![work_plane_count(world)];
    choose_plane_menu_item(world, "Offset from Plane");
}

#[given("the command is waiting for a planar face")]
fn command_waiting_for_planar_face(world: &mut ApplicationWorld) {
    world.expect_text("Offset Plane");
    world.expect_text("Select Plane");
}

#[then("the MCAD Software returns to the ready state without creating another plane")]
fn returns_to_ready_without_plane(world: &mut ApplicationWorld) {
    assert!(
        super::super::diagnostics::window_geometry("Offset Plane").is_none(),
        "expected Offset Plane dialog to close on Escape"
    );
    world.expect_text("Ready");
    let before = world.observed_counts.first().copied().unwrap_or(0);
    assert_eq!(work_plane_count(world), before);
}

#[given("three non-collinear vertices on the rectangular solid are selectable")]
fn three_vertices_selectable(world: &mut ApplicationWorld) {
    orient_isometric_for_picking(world);
    world.observed_counts = vec![work_plane_count(world)];
    let _ = three_non_collinear_vertex_points(world);
}

#[when("the user chooses Three Points and selects the three vertices")]
fn choose_three_points_and_select_vertices(world: &mut ApplicationWorld) {
    let pts = three_non_collinear_vertex_points(world);
    choose_plane_menu_item(world, "Three Points");
    for pt in pts {
        click_model_point(world, pt);
    }
}

#[then("a new work plane appears in the model browser")]
fn new_work_plane_appears(world: &mut ApplicationWorld) {
    let before = world.observed_counts.first().copied().unwrap_or(0);
    assert!(
        work_plane_count(world) > before,
        "expected a new work plane (before={before}, after={})",
        work_plane_count(world)
    );
    world.expect_text("Work Plane1");
}

#[given("an origin plane and a vertex of the rectangular solid are selectable")]
fn origin_plane_and_vertex_selectable(world: &mut ApplicationWorld) {
    orient_isometric_for_picking(world);
    ensure_origin_expanded(world);
    world.observed_counts = vec![work_plane_count(world)];
    let _ = three_non_collinear_vertex_points(world)[0];
}

#[when("the user chooses Parallel to Plane through Point")]
fn choose_parallel_to_plane_through_point(world: &mut ApplicationWorld) {
    choose_plane_menu_item(world, "Parallel to Plane through Point");
}

#[when("the user selects the origin plane and the vertex")]
fn select_origin_plane_and_vertex(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.click_text("XY Plane");
    world.pump(Vec::new());
    let vertex = three_non_collinear_vertex_points(world)[0];
    click_model_point(world, vertex);
}

#[given("an origin axis and a vertex of the rectangular solid are selectable")]
fn origin_axis_and_vertex_selectable(world: &mut ApplicationWorld) {
    orient_isometric_for_picking(world);
    ensure_origin_expanded(world);
    world.observed_counts = vec![work_plane_count(world)];
    let _ = three_non_collinear_vertex_points(world)[0];
}

#[when("the user chooses Normal to Axis through Point")]
fn choose_normal_to_axis_through_point(world: &mut ApplicationWorld) {
    choose_plane_menu_item(world, "Normal to Axis through Point");
}

#[when("the user selects the origin axis and the vertex")]
fn select_origin_axis_and_vertex(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.click_text("Z Axis");
    world.pump(Vec::new());
    let vertex = three_non_collinear_vertex_points(world)[0];
    click_model_point(world, vertex);
}

#[given(
    "the XY origin plane and the straight base edge from (0, 0, 0) to (-40, 0, 0) mm are available"
)]
fn xy_plane_and_base_edge_available(world: &mut ApplicationWorld) {
    orient_isometric_for_picking(world);
    ensure_origin_expanded(world);
    world.observed_counts = vec![work_plane_count(world)];
    let _ = base_edge_midpoint_near_origin_neg_x(world);
}

#[when("the user chooses Angle to Plane around Edge and selects those references")]
fn choose_angle_and_select_references(world: &mut ApplicationWorld) {
    let edge = base_edge_midpoint_near_origin_neg_x(world);
    choose_plane_menu_item(world, "Angle to Plane around Edge");
    ensure_origin_expanded(world);
    world.click_text("XY Plane");
    world.pump(Vec::new());
    click_model_point(world, edge);
}

#[then("an angle field appears with 90 degrees selected")]
fn angle_field_with_90_degrees_appears(world: &mut ApplicationWorld) {
    world.expect_text("Angle");
    world.expect_text("90 degrees");
    assert!(
        world.state().work_plane_preview_active,
        "expected angled plane live preview"
    );
}

#[when("the user enters 45 degrees and presses Enter")]
fn enter_45_degrees_and_press_enter(world: &mut ApplicationWorld) {
    world.click_text("90 degrees");
    world.key(ViewportPhysicalKey::A, true);
    world.type_text("45 degrees");
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Enter, false);
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[then("a new angled work plane appears in the model browser")]
fn new_angled_work_plane_appears(world: &mut ApplicationWorld) {
    let before = world.observed_counts.first().copied().unwrap_or(0);
    assert!(
        work_plane_count(world) > before,
        "expected a new angled work plane in model browser"
    );
    world.expect_text("Work Plane1");
}

#[given("two adjoining edges on a face of the rectangular solid are selectable")]
fn two_adjoining_edges_selectable(world: &mut ApplicationWorld) {
    orient_isometric_for_picking(world);
    world.observed_counts = vec![work_plane_count(world)];
    let _ = two_adjoining_edge_points(world);
}

#[when("the user chooses Two Coplanar Edges and selects those edges")]
fn choose_two_coplanar_edges_and_select(world: &mut ApplicationWorld) {
    let edges = two_adjoining_edge_points(world);
    choose_plane_menu_item(world, "Two Coplanar Edges");
    for edge in edges {
        click_model_point(world, edge);
    }
}

#[when("the user chooses Plane from the Plane menu")]
fn choose_general_plane_from_menu(world: &mut ApplicationWorld) {
    choose_plane_menu_item(world, "Plane");
}

#[then("the MCAD Software asks me to define the work plane by selecting geometry")]
fn asks_to_define_plane_by_selecting_geometry(world: &mut ApplicationWorld) {
    world.expect_text("Define work plane by selecting geometry");
}

#[when("the user chooses Midplane of Torus")]
fn choose_midplane_of_torus(world: &mut ApplicationWorld) {
    choose_plane_menu_item(world, "Midplane of Torus");
}

#[then("the MCAD Software prompts me to select a torus")]
fn prompts_to_select_torus(world: &mut ApplicationWorld) {
    world.expect_text("Select a torus");
}

#[when("the user chooses Tangent to Surface through Edge")]
fn choose_tangent_through_edge(world: &mut ApplicationWorld) {
    choose_plane_menu_item(world, "Tangent to Surface through Edge");
}

#[then("the MCAD Software prompts me to select a surface or edge")]
fn prompts_to_select_surface_or_edge(world: &mut ApplicationWorld) {
    world.expect_text("Select a surface or edge");
}

#[when("the user cancels and chooses Tangent to Surface through Point")]
fn cancel_and_choose_tangent_through_point(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    choose_plane_menu_item(world, "Tangent to Surface through Point");
}

#[then("the MCAD Software prompts me to select a surface or point")]
fn prompts_to_select_surface_or_point(world: &mut ApplicationWorld) {
    world.expect_text("Select a surface or point");
}

#[given("Work Plane2 is visible in the part browser")]
fn work_plane_2_visible_in_browser(world: &mut ApplicationWorld) {
    world.expect_text("Work Plane2");
    let browser = world.state().browser.expect("expected browser");
    let wp2 = browser
        .features
        .iter()
        .find(|f| f.name == "Work Plane2")
        .expect("expected Work Plane2 in browser");
    assert!(
        !browser.hidden_datums.contains(&wp2.id),
        "expected Work Plane2 to start visible"
    );
}

#[when("the user opens its context menu and clicks the checked Visibility command")]
fn open_wp2_context_menu_and_uncheck_visibility(world: &mut ApplicationWorld) {
    let row = world.run("Work Plane2").center();
    world.right_click(super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
    assert!(
        world.checked("Visibility"),
        "expected Visibility initially checked on Work Plane2"
    );
    world.click_text("Visibility");
    world.pump(Vec::new());
}

#[when("the user reopens the menu")]
fn reopen_wp2_context_menu(world: &mut ApplicationWorld) {
    let row = world.run("Work Plane2").center();
    world.right_click(super::super::diagnostics::pointer_pixels(row));
    world.pump(Vec::new());
}

#[when("the user clicks Visibility and reopens the menu")]
fn click_visibility_and_reopen_menu(world: &mut ApplicationWorld) {
    world.click_text("Visibility");
    world.pump(Vec::new());
    reopen_wp2_context_menu(world);
}

#[then("Visibility is checked again")]
fn visibility_checked_again(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Visibility"),
        "expected Visibility checked again on Work Plane2"
    );
}
