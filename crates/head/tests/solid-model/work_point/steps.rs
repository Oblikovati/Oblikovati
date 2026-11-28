// SPDX-License-Identifier: GPL-2.0-only
//! Desktop Wayland BDD bindings for Work Point scenarios (work-point-01..work-point-08, #3918).
mod setup;

use super::super::*;
use cucumber::{given, then, when};
use setup::{
    GROUNDED_WINDOW, choose_point_menu_item, click_browser_row, click_circular_end_edge,
    click_solid_vertex, ensure_origin_expanded, open_work_point_fixture,
};

#[given(
    "a writable copy of `fixtures/cad/docking-visible-solid-current.opd` is open with its rectangular solid visible"
)]
#[given(
    "a writable copy of `fixtures/cad/docking-visible-solid-current.opd` is open with a solid vertex visible"
)]
fn open_visible_solid_for_work_point(world: &mut ApplicationWorld) {
    open_work_point_fixture(world, "fixtures/cad/docking-visible-solid-current.opd");
    let browser = world.state().browser.expect("expected browser");
    assert_eq!(
        browser.bodies.len(),
        1,
        "expected visible rectangular solid"
    );
}

#[when("the user chooses On Vertex, Sketch point, or Midpoint")]
fn choose_on_vertex_work_point(world: &mut ApplicationWorld) {
    choose_point_menu_item(world, "On Vertex, Sketch point, or Midpoint");
}

#[then("the MCAD Software prompts for a vertex, point or midpoint")]
fn prompts_for_vertex_point_or_midpoint(world: &mut ApplicationWorld) {
    world.expect_status("Select a vertex, sketch point, or midpoint");
}

#[when("the user selects the model vertex")]
fn select_model_vertex_for_work_point(world: &mut ApplicationWorld) {
    let before = work_point_count(world);
    let vertex = click_solid_vertex(world);
    assert_eq!(
        work_point_count(world),
        before + 1,
        "expected On Vertex work point created"
    );
    let markers = world.state().work_points;
    let last = markers
        .last()
        .expect("expected evaluated work point marker");
    for axis in 0..3 {
        assert!(
            (last.position[axis] - vertex[axis]).abs() < 1.0e-3,
            "vertex mismatch on axis {axis}: got {:?}, expected {:?}",
            last.position,
            vertex
        );
    }
}

#[when("the user chooses Center Point of Loop of Edges and selects the loop")]
fn choose_loop_center_and_select_loop(world: &mut ApplicationWorld) {
    let before = work_point_count(world);
    choose_point_menu_item(world, "Center Point of Loop of Edges");
    let expected_center = click_circular_end_edge(world);
    assert_eq!(
        work_point_count(world),
        before + 1,
        "expected Loop Center work point created"
    );
    let markers = world.state().work_points;
    let last = markers
        .last()
        .expect("expected evaluated loop center marker");
    for axis in 0..3 {
        assert!(
            (last.position[axis] - expected_center[axis]).abs() < 1.0e-2,
            "loop center mismatch on axis {axis}: got {:?}, expected {:?}",
            last.position,
            expected_center
        );
    }
}

#[when("the user chooses Grounded Point and selects the vertex")]
fn choose_grounded_point_and_select_vertex(world: &mut ApplicationWorld) {
    choose_point_menu_item(world, "Grounded Point");
    click_solid_vertex(world);
}

#[then("coordinates, a move triad and accepts and cancels controls appear")]
fn grounded_coordinates_triad_and_controls_appear(world: &mut ApplicationWorld) {
    let (hud_pos, hud_size) = super::super::diagnostics::window_geometry(GROUNDED_WINDOW)
        .unwrap_or_else(|| panic!("expected Grounded Point HUD window, saw {}", world.text()));
    let text = super::super::world::squeezed(&world.window_text(GROUNDED_WINDOW));
    assert!(
        text.contains('X')
            && text.contains('Y')
            && text.contains('Z')
            && text.contains("OK")
            && text.contains("Cancel"),
        "expected X/Y/Z coordinates and OK/Cancel controls in Grounded Point HUD, saw {text}"
    );
    let grounded = world
        .state()
        .work_point
        .and_then(|wp| wp.grounded)
        .expect("expected active Grounded Point state");
    assert!(grounded.coordinates.iter().all(|v| v.is_finite()));
    assert!(
        crate::HeadWindow::model_pixel(&world.app().window.native, grounded.coordinates).is_ok(),
        "expected valid viewport projection for Grounded Point origin"
    );
    for axis_label in ["X", "Y", "Z"] {
        let triad_axis_drawn = world.frame.runs.iter().any(|run| {
            let c = run.center();
            let inside_hud = c[0] >= hud_pos[0]
                && c[0] <= hud_pos[0] + hud_size[0]
                && c[1] >= hud_pos[1]
                && c[1] <= hud_pos[1] + hud_size[1];
            run.text == axis_label && !inside_hud && c[1] > 160.0
        });
        assert!(
            triad_axis_drawn,
            "expected projected 3D move triad axis {axis_label} drawn in viewport"
        );
    }
}

#[when("the user accepts the initial position")]
fn accept_grounded_initial_position(world: &mut ApplicationWorld) {
    let before = work_point_count(world);
    let center = world.run_in_window(GROUNDED_WINDOW, "OK").center();
    world.click(super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::diagnostics::window_geometry(GROUNDED_WINDOW).is_none(),
        "expected Grounded Point HUD to close on accept"
    );
    assert_eq!(
        work_point_count(world),
        before + 1,
        "expected Grounded Point feature created"
    );
}

#[given("its X Axis and Y Axis are available in Origin")]
fn x_and_y_axes_available_in_origin(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.expect_text("X Axis");
    world.expect_text("Y Axis");
}

#[when("the user chooses Intersection of Two Lines and selects X Axis and Y Axis")]
fn choose_two_lines_and_select_x_y(world: &mut ApplicationWorld) {
    let before = work_point_count(world);
    choose_point_menu_item(world, "Intersection of Two Lines");
    click_browser_row(world, "X Axis");
    click_browser_row(world, "Y Axis");
    assert_eq!(
        work_point_count(world),
        before + 1,
        "expected Two Lines intersection work point created"
    );
    let last = world
        .state()
        .work_points
        .pop()
        .expect("expected evaluated work point");
    assert!(last.position.iter().all(|v| v.abs() < 1.0e-9));
}

#[given("its XY Plane and Z Axis are available in Origin")]
fn xy_plane_and_z_axis_available_in_origin(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.expect_text("XY Plane");
    world.expect_text("Z Axis");
}

#[when("the user chooses Intersection of Plane/Surface and Line")]
fn choose_plane_and_line_intersection(world: &mut ApplicationWorld) {
    choose_point_menu_item(world, "Intersection of Plane/Surface and Line");
}

#[when("the user selects XY Plane and Z Axis")]
fn select_xy_plane_and_z_axis(world: &mut ApplicationWorld) {
    let before = work_point_count(world);
    click_browser_row(world, "XY Plane");
    click_browser_row(world, "Z Axis");
    assert_eq!(
        work_point_count(world),
        before + 1,
        "expected Plane/Surface and Line intersection work point created"
    );
    let last = world
        .state()
        .work_points
        .pop()
        .expect("expected evaluated work point");
    assert!(last.position.iter().all(|v| v.abs() < 1.0e-9));
}

#[when("the user chooses Center Point of Torus")]
fn choose_center_point_of_torus(world: &mut ApplicationWorld) {
    choose_point_menu_item(world, "Center Point of Torus");
}

#[then("the MCAD Software prompts me to select a torus")]
fn prompts_to_select_torus(world: &mut ApplicationWorld) {
    world.expect_status("Select a torus");
}

#[when("the user cancels and chooses Center Point of Sphere")]
fn cancel_and_choose_center_point_of_sphere(world: &mut ApplicationWorld) {
    world.escape();
    world.pump(Vec::new());
    world.expect_ready();
    choose_point_menu_item(world, "Center Point of Sphere");
}

#[then("the MCAD Software prompts me to select a sphere")]
fn prompts_to_select_sphere(world: &mut ApplicationWorld) {
    world.expect_status("Select a sphere");
}

#[when("the user chooses Point from the Point menu")]
fn choose_general_point_from_menu(world: &mut ApplicationWorld) {
    choose_point_menu_item(world, "Point");
}

#[then("the MCAD Software prompts me to define the work point by selecting geometry")]
fn prompts_to_define_work_point_geometry(world: &mut ApplicationWorld) {
    world.expect_status("Define work point by selecting geometry");
}

#[given("a writable copy of `fixtures/cad/work-point/six-points.opd` is open")]
fn open_six_points_fixture(world: &mut ApplicationWorld) {
    open_work_point_fixture(world, "fixtures/cad/work-point/six-points.opd");
}

#[given("Work Point1 through Work Point6 exist")]
fn six_work_points_exist(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let names: Vec<_> = browser.features.iter().map(|f| f.name.as_str()).collect();
    assert_eq!(
        names,
        [
            "Work Point1",
            "Work Point2",
            "Work Point3",
            "Work Point4",
            "Work Point5",
            "Work Point6"
        ]
    );
}

#[given("the YZ, XZ, and XY origin planes are available in the browser")]
fn origin_planes_available_in_browser(world: &mut ApplicationWorld) {
    ensure_origin_expanded(world);
    world.expect_text("YZ Plane");
    world.expect_text("XZ Plane");
    world.expect_text("XY Plane");
}

#[when("the user chooses Intersection of Three Planes")]
fn choose_three_planes_intersection(world: &mut ApplicationWorld) {
    choose_point_menu_item(world, "Intersection of Three Planes");
}

#[when("the user selects the YZ, XZ, and XY planes in turn")]
fn select_yz_xz_xy_planes(world: &mut ApplicationWorld) {
    click_browser_row(world, "YZ Plane");
    click_browser_row(world, "XZ Plane");
    click_browser_row(world, "XY Plane");
}

#[then("a new Work Point7 appears at the end of the browser")]
fn work_point7_appears_at_end(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let last = browser
        .features
        .last()
        .expect("expected Work Point7 at end of browser");
    assert_eq!(last.name, "Work Point7");
    assert_eq!(last.kind, "Work Point");
    world.expect_text("Work Point7");
}

#[when("the user selects Work Point7")]
fn select_work_point7(world: &mut ApplicationWorld) {
    click_browser_row(world, "Work Point7");
}

#[then("its browser entry is highlighted")]
fn work_point7_browser_entry_highlighted(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected browser");
    let wp7 = browser
        .features
        .iter()
        .find(|f| f.name == "Work Point7")
        .expect("expected Work Point7 in browser");
    assert_eq!(browser.selected_feature, Some(wp7.id));
    let r7 = world.run("Work Point7").clone();
    let r6 = world.run("Work Point6").clone();
    let x = r7.maximum[0].max(r6.maximum[0]) as usize + 6;
    let p7 = world.pixel_at([x, r7.center()[1] as usize]);
    let p6 = world.pixel_at([x, r6.center()[1] as usize]);
    let p7_mid = world.pixel_at([
        (r7.minimum[0] as usize).saturating_sub(4),
        r7.center()[1] as usize,
    ]);
    let p6_mid = world.pixel_at([
        (r6.minimum[0] as usize).saturating_sub(4),
        r6.center()[1] as usize,
    ]);
    let model = super::super::diagnostics::window_geometry("Model");
    assert!(
        p7 != p6 || p7_mid != p6_mid,
        "expected Work Point7 highlighted: r7={r7:?}, r6={r6:?}, model={model:?}, p7={p7:?}, p6={p6:?}, p7_mid={p7_mid:?}, p6_mid={p6_mid:?}"
    );
}

fn work_point_count(world: &mut ApplicationWorld) -> usize {
    world
        .state()
        .browser
        .expect("expected browser")
        .features
        .iter()
        .filter(|f| f.kind == "Work Point")
        .count()
}
