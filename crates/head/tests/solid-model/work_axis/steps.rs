// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Part Work Features > Work Axis (work-axis-01..09a).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use setup::{
    click_browser_entry, click_inner_cylindrical_wall, click_model_point, ensure_origin_expanded,
    open_axis_dropdown_and_choose, open_work_axis_fixture, scene_bounds, work_axis_count,
};

#[given(
    "a writable copy of `fixtures/cad/docking-visible-solid-current.opd` is open with two distinct solid vertices visible"
)]
#[given(
    "a writable copy of `fixtures/cad/docking-visible-solid-current.opd` is open with a straight solid edge visible"
)]
fn open_visible_solid_for_work_axis(world: &mut ApplicationWorld) {
    open_work_axis_fixture(world, "fixtures/cad/docking-visible-solid-current.opd");
}

#[given("a writable copy of `fixtures/cad/blank-metric-part.opd` is open")]
#[given("`fixtures/cad/blank-metric-part.opd` is open")]
fn open_blank_metric_for_work_axis(world: &mut ApplicationWorld) {
    open_work_axis_fixture(world, "fixtures/cad/blank-metric-part.opd");
}

#[given(
    "a writable copy of `fixtures/cad/fillet-circular-edge-candidate.opd` is open with a circular end edge visible"
)]
fn open_cylinder_for_work_axis(world: &mut ApplicationWorld) {
    open_work_axis_fixture(world, "fixtures/cad/fillet-circular-edge-candidate.opd");
}

#[given(
    "a writable copy of `fixtures/cad/work-axis/hole-seven-axes-unique-candidate.opd` is open with Work Axis1 through Work Axis7 present"
)]
fn open_hole_seven_axes(world: &mut ApplicationWorld) {
    open_work_axis_fixture(
        world,
        "fixtures/cad/work-axis/hole-seven-axes-unique-candidate.opd",
    );
    assert_eq!(
        work_axis_count(world),
        7,
        "expected Work Axis1..Work Axis7 in hole fixture"
    );
}

#[given(
    "a writable copy of `fixtures/cad/work-axis/internal-wall-seven-axes.opd` is open with Work Axis1 through Work Axis7 present"
)]
fn open_internal_wall_seven_axes(world: &mut ApplicationWorld) {
    open_work_axis_fixture(world, "fixtures/cad/work-axis/internal-wall-seven-axes.opd");
    assert_eq!(
        work_axis_count(world),
        7,
        "expected Work Axis1..Work Axis7 in internal wall fixture"
    );
}

#[given("the XY Plane and XZ Plane are available in Origin")]
fn origin_xy_and_xz_available(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.expect_text("XYPlane");
    world.expect_text("XZPlane");
}

#[given("the XY Plane and a solid vertex are available")]
fn origin_xy_and_vertex_available(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.expect_text("XYPlane");
    let _ = scene_bounds(world);
}

#[given("the Z Axis and a solid vertex not on that axis are available")]
fn origin_z_and_off_axis_vertex_available(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.expect_text("ZAxis");
    let (_, max) = scene_bounds(world);
    assert!(max[0].hypot(max[1]) > 1e-4);
}

#[given("the cylindrical side surface is visible")]
fn cylindrical_side_visible(world: &mut ApplicationWorld) {
    let (min, max) = scene_bounds(world);
    assert!(max[0] > min[0] && max[2] > min[2]);
}

#[given("the cylindrical surface inside Hole1 is visible")]
#[given("the 9 mm cylindrical inside wall is visible")]
fn internal_cylindrical_surface_visible(world: &mut ApplicationWorld) {
    let (min, max) = scene_bounds(world);
    assert!(max[2] > min[2]);
}

#[when("the user chooses Through Two Points and selects both vertices")]
fn choose_through_two_points_and_select(world: &mut ApplicationWorld) {
    open_axis_dropdown_and_choose(world, "Through Two Points");
    let (min, max) = scene_bounds(world);
    click_model_point(world, [max[0], max[1], max[2]]);
    click_model_point(world, [min[0], max[1], max[2]]);
}

#[when("the user chooses On Line or Edge and selects the edge")]
fn choose_on_line_or_edge_and_select(world: &mut ApplicationWorld) {
    open_axis_dropdown_and_choose(world, "On Line or Edge");
    let (min, max) = scene_bounds(world);
    click_model_point(world, [(min[0] + max[0]) * 0.5, max[1], max[2]]);
}

#[when("the user chooses Intersection of Two Planes and selects XY Plane and XZ Plane")]
fn choose_plane_intersection_and_select(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    open_axis_dropdown_and_choose(world, "Intersection of Two Planes");
    click_browser_entry(world, "XY Plane");
    click_browser_entry(world, "XZ Plane");
}

#[when("the user chooses Through Center of Circular or Elliptical Edge and selects the edge")]
fn choose_circular_edge_and_select(world: &mut ApplicationWorld) {
    open_axis_dropdown_and_choose(world, "Through Center of Circular or Elliptical Edge");
    let (min, max) = scene_bounds(world);
    let cx = (min[0] + max[0]) * 0.5;
    let cy = (min[1] + max[1]) * 0.5;
    let radius = (max[0] - min[0]) * 0.5;
    click_model_point(world, [cx + radius, cy, max[2]]);
}

#[when("the user chooses Normal to Plane through Point and selects XY Plane and the vertex")]
fn choose_normal_to_plane_and_select(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    open_axis_dropdown_and_choose(world, "Normal to Plane through Point");
    click_browser_entry(world, "XY Plane");
    let (_, max) = scene_bounds(world);
    click_model_point(world, [max[0], max[1], max[2]]);
}

#[when("the user chooses Parallel to Line through Point")]
fn choose_parallel_to_line(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    open_axis_dropdown_and_choose(world, "Parallel to Line through Point");
}

#[when("the user selects Z Axis and the vertex")]
fn select_z_axis_and_vertex(world: &mut ApplicationWorld) {
    click_browser_entry(world, "Z Axis");
    let (_, max) = scene_bounds(world);
    click_model_point(world, [max[0], max[1], max[2]]);
}

#[when("the user chooses Through Revolved Face or Feature and selects the surface")]
fn choose_revolved_face_and_select_outer_surface(world: &mut ApplicationWorld) {
    open_axis_dropdown_and_choose(world, "Through Revolved Face or Feature");
    let (min, max) = scene_bounds(world);
    let cx = (min[0] + max[0]) * 0.5;
    let cy = (min[1] + max[1]) * 0.5;
    let radius = (max[0] - min[0]) * 0.5;
    let mid_z = (min[2] + max[2]) * 0.5;
    let angle = -std::f64::consts::FRAC_PI_4;
    click_model_point(
        world,
        [cx + radius * angle.cos(), cy + radius * angle.sin(), mid_z],
    );
}

#[when("the user chooses Axis from the Axis menu")]
fn choose_general_axis_from_menu(world: &mut ApplicationWorld) {
    open_axis_dropdown_and_choose(world, "Axis");
}

#[when("the user chooses Through Revolved Face or Feature")]
fn choose_through_revolved_face(world: &mut ApplicationWorld) {
    open_axis_dropdown_and_choose(world, "Through Revolved Face or Feature");
}

#[when("the user clicks Hole1 in the browser")]
fn click_hole1_in_browser(world: &mut ApplicationWorld) {
    click_browser_entry(world, "Hole1");
}

#[when("the user selects the curved surface inside the hole")]
fn select_curved_surface_inside_hole(world: &mut ApplicationWorld) {
    click_inner_cylindrical_wall(world, 0.15);
}

#[when("the user selects the curved inside wall")]
fn select_curved_inside_wall(world: &mut ApplicationWorld) {
    click_inner_cylindrical_wall(world, 0.45);
}

#[when("the user selects Work Axis8")]
fn select_work_axis8(world: &mut ApplicationWorld) {
    click_browser_entry(world, "Work Axis8");
}

#[then("a work axis appears in the model browser")]
#[then("a new work axis appears in the model browser")]
fn work_axis_appears_in_browser(world: &mut ApplicationWorld) {
    assert!(
        work_axis_count(world) >= 1,
        "expected at least one Work Axis in browser, saw {:?}",
        world.state().browser
    );
    world.expect_text("WorkAxis1");
    world.expect_ready();
}

#[then("the MCAD Software returns to Ready and displays an axis")]
fn returns_to_ready_and_displays_axis(world: &mut ApplicationWorld) {
    world.expect_ready();
    let bounds = crate::HeadWindow::presentation_bounds(&world.app().window.native);
    let axes = world.app().owner.work_axis_displays(bounds);
    let datum = world.app().owner.datum_planes().unwrap();
    assert!(
        !axes.is_empty() && !datum.axes.is_empty(),
        "expected a displayed work axis overlay (projected={axes:?}, 3d={:?})",
        datum.axes
    );
    assert!(
        work_axis_count(world) >= 1,
        "expected a work axis feature in browser"
    );
}

#[then("the MCAD Software prompts me to define the axis by selecting geometry")]
fn prompts_to_define_axis(world: &mut ApplicationWorld) {
    let prompt = world
        .state()
        .work_axis_prompt
        .expect("expected active Work Axis prompt");
    assert!(
        prompt.to_lowercase().contains("define") && prompt.to_lowercase().contains("geometry"),
        "unexpected general Axis prompt: {prompt:?}"
    );
    world.expect_status("Define work axis by selecting geometry");
}

#[then("the MCAD Software continues to prompt for a cylindrical or revolved surface")]
fn continues_prompting_for_revolved_surface(world: &mut ApplicationWorld) {
    let prompt = world
        .state()
        .work_axis_prompt
        .expect("expected active revolved-surface prompt after clicking Hole1");
    assert!(
        prompt.contains("cylindrical or revolved surface"),
        "unexpected revolved-surface prompt: {prompt:?}"
    );
    world.expect_status("Select a cylindrical or revolved surface for work axis");
}

#[then("the MCAD Software returns to Ready and Work Axis8 appears in the browser")]
fn returns_to_ready_and_work_axis8_appears(world: &mut ApplicationWorld) {
    world.expect_ready();
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .features
            .iter()
            .any(|feature| feature.name == "Work Axis8" && feature.kind == "Work Axis"),
        "expected Work Axis8 in browser, saw {:?}",
        browser.features
    );
    world.expect_text("WorkAxis8");
}

#[then("an axis through the hole is highlighted")]
#[then("an axis through the internal wall is highlighted")]
fn work_axis8_is_highlighted(world: &mut ApplicationWorld) {
    let bounds = crate::HeadWindow::presentation_bounds(&world.app().window.native);
    let axes = world.app().owner.work_axis_displays(bounds);
    let highlighted = axes
        .iter()
        .find(|axis| axis.name == "Work Axis8")
        .unwrap_or_else(|| panic!("expected Work Axis8 overlay, saw {axes:?}"));
    assert!(
        highlighted.selected,
        "expected Work Axis8 to be highlighted when selected, saw {highlighted:?}"
    );
    let (min, max) = scene_bounds(world);
    let cx = (min[0] + max[0]) * 0.5;
    let cy = (min[1] + max[1]) * 0.5;
    let [p0, p1] = highlighted.endpoints;
    assert!(
        (p0[0] - cx).abs() < 1e-3
            && (p0[1] - cy).abs() < 1e-3
            && (p1[0] - cx).abs() < 1e-3
            && (p1[1] - cy).abs() < 1e-3,
        "expected Work Axis8 along hole/internal-wall Z centerline ({cx}, {cy}), saw {p0:?}..{p1:?}"
    );
    let s0 = crate::HeadWindow::model_pixel(&world.app().window.native, p0).unwrap();
    let s1 = crate::HeadWindow::model_pixel(&world.app().window.native, p1).unwrap();
    let dx = s1[0] - s0[0];
    let dy = s1[1] - s0[1];
    assert!(
        dx * dx + dy * dy > 16,
        "expected visible non-degenerate highlighted axis segment, saw {s0:?}..{s1:?}"
    );
    let datum = world.app().owner.datum_planes().unwrap();
    assert!(
        datum.axes.iter().any(|axis| axis.selected),
        "expected selected 3D ViewportDatumAxis highlight, saw {:?}",
        datum.axes
    );
}
